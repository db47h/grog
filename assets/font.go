package assets

import (
	"io/ioutil"
	"path"

	"github.com/db47h/grog/text"
	"github.com/db47h/grog/texture"
	"github.com/db47h/ofs"
	"github.com/golang/freetype/truetype"
	"github.com/pkg/errors"
	"golang.org/x/image/font"
)

type fnt struct {
	name string
	f    *truetype.Font
	ds   map[fntOpts]*text.Drawer
}

func (f *fnt) close() error {
	errs := make(errorList)
	for opts, d := range f.ds {
		if err := d.Face().Close(); err != nil {
			// only one face error per font
			errs[f.name] = errors.Wrapf(err, "face %v", opts)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

type fntOpts struct {
	sz float64
	h  text.Hinting
	mf texture.FilterMode
}

// FontPath returns an Option that sets the default font path.
//
func FontPath(name string) Option {
	return cfn(func(cfg *config) {
		cfg.fontPath = name
	})
}

func loadFont(fs ofs.FileSystem, name string) (asset, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	ttf, err := truetype.Parse(data)
	if err != nil {
		return nil, err
	}
	return &fnt{name, ttf, make(map[fntOpts]*text.Drawer)}, nil
}

func (m *Manager) PreloadFont(name string) {
	name = path.Join(m.cfg.fontPath, name)
	if !m.loadStart(name) {
		return
	}
	m.cs <- func() {
		a, err := loadFont(m.fs, name)
		if err != nil {
			m.loadError(name, err)
		} else {
			m.loadComplete(name, a)
		}
	}
}

// Font returns the named font asset.
//
func (m *Manager) Font(name string) (*truetype.Font, error) {
	name = path.Join(m.cfg.fontPath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		a, err, s := m.assetNoLock(name)
		switch s {
		case stateMissing:
			a, err = m.syncLoadNoLock(name, loadFont)
			if err != nil {
				return nil, err
			}
			fallthrough
		case stateLoaded:
			f, ok := a.(*fnt)
			if !ok {
				return nil, errors.Errorf("asset %s is not a font", name)
			}
			return f.f, nil
		case stateError:
			return nil, err
		}
		m.cond.Wait()
	}
}

// FontDrawer returns a new text.Drawer configured for the given font face (with
// a default DPI of 72).
//
// Note that this function caches any text.Drawer created. The only way to clean
// the cache is to Dispose() the corresponding font asset. If an application
// needs to be able to discard drawers, it should use Font() instead and manage
// font.Face and text.Drawer creation and caching manually.
//
func (m *Manager) FontDrawer(name string, size float64, hinting text.Hinting, magFilter texture.FilterMode) (*text.Drawer, error) {
	name = path.Join(m.cfg.fontPath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		a, err, s := m.assetNoLock(name)
		switch s {
		case stateMissing:
			a, err = m.syncLoadNoLock(name, loadFont)
			if err != nil {
				return nil, err
			}
			fallthrough
		case stateLoaded:
			f, ok := a.(*fnt)
			if !ok {
				return nil, errors.Errorf("asset %s is not a font", name)
			}
			opts := fntOpts{size, hinting, magFilter}
			if ff := f.ds[opts]; ff != nil {
				return ff, nil
			}
			ff := text.NewDrawer(truetype.NewFace(f.f, &truetype.Options{
				Size:       size,
				Hinting:    font.Hinting(hinting),
				DPI:        72,
				SubPixelsX: text.SubPixelsX,
			}), magFilter)
			f.ds[opts] = ff
			return ff, nil
		case stateError:
			return nil, err
		}
		m.cond.Wait()
	}
}

// DiscardFont removes the named font from the asset cache along with any associated resources.
//
func (m *Manager) DiscardFont(name string) error {
	return m.discard(path.Join(m.cfg.fontPath, name))
}
