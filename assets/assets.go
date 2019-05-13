package assets

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"path"
	"strings"
	"sync"

	"github.com/db47h/grog/text"
	"github.com/db47h/grog/texture"
	"github.com/db47h/ofs"
	"github.com/golang/freetype/truetype"
	"github.com/pkg/errors"
	"golang.org/x/image/font"
)

var errMissingAsset = errors.New("asset not found")

type errorList map[string]error

func (e errorList) Error() string {
	var sb strings.Builder
	i := 0
	for k, err := range e {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.Write([]byte{':', ' '})
		sb.WriteString(err.Error())
		i++
	}
	return sb.String()
}

type cmd int

const (
	cmdLoadTexture cmd = iota
	cmdLoadFont
)

type pending struct {
	cmd
	name string
}

type tex struct {
	img    image.Image
	params []texture.ParameterFunc
}

type Manager struct {
	fs     ofs.FileSystem
	cfg    *Config
	m      sync.Mutex
	cond   *sync.Cond
	errs   errorList
	assets map[string]interface{}
	ps     map[pending]struct{}
	cs     chan func()
}

type Config struct {
	TexturePath string
	FontPath    string
}

func NewManager(fs ofs.FileSystem, cfg *Config) *Manager {
	if cfg == nil {
		cfg = new(Config)
	}
	m := &Manager{
		fs:     fs,
		cfg:    cfg,
		errs:   make(errorList),
		assets: make(map[string]interface{}),
		ps:     make(map[pending]struct{}),
		cs:     make(chan func(), 4096),
	}
	m.cond = sync.NewCond(&m.m)
	for i := 0; i < 8; i++ {
		go func() {
			for f := range m.cs {
				f() // f must remove itself from ps
			}
		}()
	}
	return m
}

func (m *Manager) error(cmd cmd, name string, err error) {
	m.m.Lock()
	m.errs[name] = err
	delete(m.ps, pending{cmd, name})
	m.cond.Broadcast()
	m.m.Unlock()
}

func (m *Manager) errForAssetNoLock(name string) error {
	if err, ok := m.errs[name]; ok {
		return errors.Wrap(err, name)
	}
	return errors.Wrap(errMissingAsset, name)
}

func (m *Manager) Errors() error {
	if len(m.errs) == 0 {
		return nil
	}
	return m.errs
}

func (m *Manager) cmdStart(cmd cmd, name string) (ok bool) {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.ps[pending{cmd, name}]; ok {
		return false
	}
	if _, ok := m.assets[name]; ok {
		return false
	}
	m.ps[pending{cmd, name}] = struct{}{}
	return true
}

func (m *Manager) cmdCompleteNoLock(cmd cmd, name string) {
	delete(m.ps, pending{cmdLoadTexture, name})
	m.cond.Broadcast()
}

func (m *Manager) cmdInProgressNoLock(cmd cmd, name string) bool {
	_, ok := m.ps[pending{cmd, name}]
	return ok
}

func (m *Manager) LoadTexture(name string, params ...texture.ParameterFunc) {
	name = path.Join(m.cfg.TexturePath, name)
	if !m.cmdStart(cmdLoadTexture, name) {
		return
	}

	m.cs <- func() {
		r, err := m.fs.Open(name)
		if err != nil {
			m.error(cmdLoadTexture, name, err)
			return
		}
		src, _, err := image.Decode(r)
		if err != nil {
			m.error(cmdLoadTexture, name, err)
			return
		}
		// update
		m.m.Lock()
		m.assets[name] = &tex{src, params}
		m.cmdCompleteNoLock(cmdLoadTexture, name)
		m.m.Unlock()
	}
}

func (m *Manager) Texture(name string) (*texture.Texture, error) {
	name = path.Join(m.cfg.TexturePath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		// always check textures first, just in case someone else built it
		// which is very unlikely since this function should be called from the main thread
		t, ok := m.assets[name]
		if ok {
			switch t := t.(type) {
			case *texture.Texture:
				return t, nil
			case *tex:
				tx := texture.New(t.img, t.params...)
				m.assets[name] = tx
				return tx, nil
			default:
				return nil, errors.Errorf("asset %s is not a texture", name)
			}
		}
		if !m.cmdInProgressNoLock(cmdLoadTexture, name) {
			// not found. Check if we have any error for this one
			return nil, m.errForAssetNoLock(name)
		}
		m.cond.Wait()
	}
}

func (m *Manager) QueueSize() int {
	m.m.Lock()
	s := len(m.ps)
	m.m.Unlock()
	return s
}

type fnt truetype.Font

func (f *fnt) NewFace(size float64, hinting font.Hinting) (font.Face, error) {
	return truetype.NewFace((*truetype.Font)(f), &truetype.Options{Size: size, DPI: 72, Hinting: hinting}), nil
}

func (m *Manager) LoadFont(name string) {
	name = path.Join(m.cfg.FontPath, name)
	if !m.cmdStart(cmdLoadFont, name) {
		return
	}
	m.cs <- func() {
		f, err := m.fs.Open(name)
		if err != nil {
			m.error(cmdLoadFont, name, err)
			return
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			m.error(cmdLoadFont, name, err)
			return
		}
		ttf, err := truetype.Parse(data)
		if err != nil {
			m.error(cmdLoadFont, name, err)
			return
		}
		m.m.Lock()
		m.assets[name] = (*fnt)(ttf)
		m.cmdCompleteNoLock(cmdLoadFont, name)
		m.m.Unlock()
	}
}

func (m *Manager) Font(name string) (text.Facer, error) {
	name = path.Join(m.cfg.FontPath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		if f, ok := m.assets[name]; ok {
			if fr, ok := f.(text.Facer); ok {
				return fr, nil
			}
			return nil, errors.Errorf("asset %s is not a font", name)
		}
		if !m.cmdInProgressNoLock(cmdLoadFont, name) {
			return nil, m.errForAssetNoLock(name)
		}
		m.cond.Wait()
	}
}
