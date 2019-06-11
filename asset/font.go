package asset

import (
	"io/ioutil"

	"github.com/db47h/grog"
	"github.com/db47h/ofs"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/xerrors"
)

type fnt struct {
	name string
	f    *truetype.Font
	ds   map[fntOpts]*grog.TextDrawer
}

func (f *fnt) Close() error {
	var errs errorList
	for opts, d := range f.ds {
		if err := d.Face().Close(); err != nil {
			errs = append(errs, xerrors.Errorf("close face %v: %w", opts, err))
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

type fntOpts struct {
	sz float64
	h  grog.Hinting
	mf grog.TextureFilter
}

// FontPath returns an Option that sets the default font path.
//
func FontPath(name string) Option {
	return cfn(func(cfg *config) {
		cfg.fontPath = name
	})
}

func loadFont(fs ofs.FileSystem, name string) (interface{}, error) {
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
	return &fnt{name, ttf, make(map[fntOpts]*grog.TextDrawer)}, nil
}

// Font returns the named font asset.
//
func (m *Manager) Font(name string) (*truetype.Font, error) {
	m.m.Lock()
	defer m.m.Unlock()
	for {
		var err error
		a, s := m.getAsset(TypeFont, name)
		switch s {
		case stateMissing:
			a, err = m.syncLoad(TypeFont, name, loadFont)
			if err != nil {
				return nil, err
			}
			fallthrough
		case stateLoaded:
			f, ok := a.(*fnt)
			if !ok {
				return nil, xerrors.Errorf("asset %s is not a font", name)
			}
			return f.f, nil
		}
		m.cond.Wait()
	}
}

// TextDrawer returns a new grog.TextDrawer configured for the given font face (with
// a default DPI of 72).
//
// Note that this function caches any grog.TextDrawer created. The only way to clean
// the cache is to Dispose() the corresponding font asset. If an application
// needs to be able to discard drawers, it should use Font() instead and manage
// font.Face and grog.Drawer creation and caching manually.
//
func (m *Manager) TextDrawer(name string, size float64, hinting grog.Hinting, magFilter grog.TextureFilter) (*grog.TextDrawer, error) {
	m.m.Lock()
	defer m.m.Unlock()
	for {
		var err error
		a, s := m.getAsset(TypeFont, name)
		switch s {
		case stateMissing:
			a, err = m.syncLoad(TypeFont, name, loadFont)
			if err != nil {
				return nil, err
			}
			fallthrough
		case stateLoaded:
			f, ok := a.(*fnt)
			if !ok {
				return nil, xerrors.Errorf("asset %s is not a font", name)
			}
			opts := fntOpts{size, hinting, magFilter}
			if ff := f.ds[opts]; ff != nil {
				return ff, nil
			}
			ff := grog.NewTextDrawer(truetype.NewFace(f.f, &truetype.Options{
				Size:       size,
				Hinting:    font.Hinting(hinting),
				DPI:        72,
				SubPixelsX: grog.FontSubPixelsX,
				SubPixelsY: grog.FontSubPixelsY,
			}), magFilter)
			f.ds[opts] = ff
			return ff, nil
		}
		m.cond.Wait()
	}
}
