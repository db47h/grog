package asset

import (
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/db47h/ofs"
	"golang.org/x/xerrors"
)

var errMissingAsset = xerrors.New("asset not found")

type errorList map[Asset]error

func (e errorList) Error() string {
	var sb strings.Builder
	i := 0
	for k, err := range e {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k.String())
		sb.Write([]byte{':', ' '})
		sb.WriteString(err.Error())
		i++
	}
	return sb.String()
}

type Closer interface {
	Close() error
}

// A Manager manages asynchronous (pre)loading and caching of textures, fonts an raw files.
//
type Manager struct {
	fs      ofs.FileSystem
	cfg     *config
	m       sync.Mutex
	cond    *sync.Cond
	assets  map[Asset]interface{}
	pending map[Asset]struct{}
}

type config struct {
	texturePath string
	fontPath    string
	filePath    string
}

// Option is implemented by option functions passed as arguments to NewManager.
//
type Option interface {
	set(*config)
}

type cfn func(*config)

func (f cfn) set(cfg *config) {
	f(cfg)
}

// NewManager returns a new asset Manager.
//
func NewManager(fs ofs.FileSystem, options ...Option) *Manager {
	cfg := new(config)
	for _, o := range options {
		o.set(cfg)
	}

	m := &Manager{
		fs:      fs,
		cfg:     cfg,
		assets:  make(map[Asset]interface{}),
		pending: make(map[Asset]struct{}),
	}
	m.cond = sync.NewCond(&m.m)
	return m
}

type loadState int

const (
	stateMissing = iota
	statePending
	stateLoaded
)

func (m *Manager) getAsset(t Type, name string) (a interface{}, state loadState) {
	k := Asset{t, name}
	if a, ok := m.assets[k]; ok {
		return a, stateLoaded
	}
	if _, ok := m.pending[k]; ok {
		return nil, statePending
	}
	return nil, stateMissing
}

func (m *Manager) syncLoad(t Type, name string, f func(fs ofs.FileSystem, name string) (interface{}, error)) (interface{}, error) {
	k := Asset{t, name}
	m.pending[k] = struct{}{}
	name = m.assetPath(&k)
	m.m.Unlock()
	a, err := f(m.fs, name)
	m.m.Lock()
	delete(m.pending, k)
	if err != nil {
		return nil, xerrors.Errorf("%s: %w", name, err)
	}
	m.assets[k] = a
	return a, nil
}

func (m *Manager) Discard(a Asset) error {
	m.m.Lock()
	for {
		if aa, ok := m.assets[a]; ok {
			delete(m.assets, a)
			m.m.Unlock()
			if cl, ok := aa.(Closer); ok {
				return cl.Close()
			}
			return nil
		}
		if _, ok := m.pending[a]; !ok {
			m.m.Unlock()
			return xerrors.Errorf("%s: %w", a, errMissingAsset)
		}
		m.cond.Wait()
	}
}

func (m *Manager) wait() {
	for len(m.pending) > 0 {
		m.cond.Wait()
	}
}

func (m *Manager) Wait(rc <-chan Result) error {
	errs := make(errorList)
	for r := range rc {
		if r.Err != nil {
			errs[r.Asset] = r.Err
		}
	}
	if len(errs) != 0 {
		return errs
	}
	return nil
}

// Close discards all assets and stops any spawned goroutines. Any subsequent
// call to a Load function will cause a panic.
//
func (m *Manager) Close() error {
	m.m.Lock()
	m.wait()
	errs := make(errorList)
	for k, a := range m.assets {
		if cl, ok := a.(Closer); ok {
			if err := cl.Close(); err != nil {
				errs[k] = err
			}
		}
	}
	m.assets = nil
	if len(errs) != 0 {
		return errs
	}
	return nil
}

type Type int

const (
	TypeFont = iota
	TypeTexture
	TypeFile
)

type Asset struct {
	Type
	Name string
}

func (a *Asset) String() string {
	switch a.Type {
	case TypeFont:
		return "font " + a.Name
	case TypeTexture:
		return "texture " + a.Name
	case TypeFile:
		return "file " + a.Name
	}
	return "unknown asset type " + a.Name
}

type Result struct {
	Asset
	Err error
}

func Font(name string) Asset    { return Asset{TypeFont, name} }
func Texture(name string) Asset { return Asset{TypeTexture, name} }
func File(name string) Asset    { return Asset{TypeFile, name} }

func (m *Manager) assetPath(a *Asset) string {
	switch a.Type {
	case TypeFont:
		return path.Join(m.cfg.fontPath, a.Name)
	case TypeTexture:
		return path.Join(m.cfg.texturePath, a.Name)
	case TypeFile:
		return path.Join(m.cfg.filePath, a.Name)
	}
	return a.Name
}

// Preload bulk preloads assets. If the discardUnused argument is true, cached
// assets not present in the asset list will be removed from the cache. It
// returns a channel to read preload results from as well as the number of items
// that are actually being preloaded.
//
// While preload starts immediately, preloading will stall after a few assets
// have been preloaded until the rc channel is read from (or Wait is called).
//
func (m *Manager) Preload(assets []Asset, discardUnused bool) (rc <-chan Result, expected int) {
	m.m.Lock()
	if discardUnused {
		amap := map[Asset]struct{}{}
		for i := range assets {
			amap[assets[i]] = struct{}{}
		}
		m.wait()
		for k := range m.assets {
			if _, ok := amap[k]; !ok {
				delete(m.assets, k)
			}
		}
	}

	// mark assets as pending and ignore loaded/pending assets
	for i := len(assets) - 1; i > 0; i-- {
		a := &assets[i]
		_, state := m.getAsset(a.Type, a.Name)
		if state != stateMissing {
			copy(assets[i:], assets[i+1:])
			assets = assets[:len(assets)-1]
			continue
		}
		m.pending[*a] = struct{}{}
	}
	m.m.Unlock()

	c := make(chan Result)
	go m.preload(assets, c)
	return c, len(assets)
}

func (m *Manager) preload(assets []Asset, rc chan Result) {
	wg := new(sync.WaitGroup)
	c := make(chan func())
	for i := 0; i < 2*runtime.NumCPU(); i++ {
		go func(c chan func()) {
			for f := range c {
				f()
			}
		}(c)
	}

	wg.Add(len(assets))

	for i := range assets {
		a := assets[i]
		c <- func() {
			var (
				data interface{}
				err  error
				name = m.assetPath(&a)
			)
			switch a.Type {
			case TypeFont:
				data, err = loadFont(m.fs, name)
			case TypeTexture:
				data, err = loadTexture(m.fs, name)
			case TypeFile:
				data, err = loadFile(m.fs, name)
			default:
				panic("Unknown asset type")
			}
			m.m.Lock()
			delete(m.pending, a)
			if err == nil {
				m.assets[a] = data
			}
			m.cond.Broadcast()
			m.m.Unlock()
			rc <- Result{Asset: a, Err: err}
			wg.Done()
		}
	}
	close(c)
	wg.Wait()
	close(rc)
}
