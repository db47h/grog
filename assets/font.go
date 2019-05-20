package assets

import (
	"io/ioutil"
	"path"

	"github.com/db47h/grog/text"
	"github.com/db47h/grog/texture"
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

func (m *Manager) LoadFont(name string) {
	name = path.Join(m.cfg.fontPath, name)
	if !m.loadStart(name) {
		return
	}
	m.cs <- func() {
		f, err := m.fs.Open(name)
		if err != nil {
			m.error(name, err)
			return
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			m.error(name, err)
			return
		}
		ttf, err := truetype.Parse(data)
		if err != nil {
			m.error(name, err)
			return
		}
		m.loadComplete(name, &fnt{name, ttf, make(map[fntOpts]*text.Drawer)})
	}
}

// Font returns the named font asset.
//
func (m *Manager) Font(name string) (*truetype.Font, error) {
	name = path.Join(m.cfg.fontPath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		if a, ok := m.assets[name]; ok {
			f, ok := a.(*fnt)
			if !ok {
				return nil, errors.Errorf("asset %s is not a font", name)
			}
			return f.f, nil
		}
		if !m.loadInProgressNoLock(name) {
			return nil, m.errForAssetNoLock(name)
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
		if v, ok := m.assets[name]; ok {
			f, ok := v.(*fnt)
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
		}
		if !m.loadInProgressNoLock(name) {
			return nil, m.errForAssetNoLock(name)
		}
		m.cond.Wait()
	}
}
