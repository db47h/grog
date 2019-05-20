package assets

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"path"

	"github.com/db47h/grog/texture"
	"github.com/db47h/ofs"
	"github.com/pkg/errors"
)

type texImage struct {
	img    image.Image
	params []texture.Parameter
}

func (*texImage) close() error { return nil }

type tex texture.Texture

func (t *tex) close() error {
	(*texture.Texture)(t).Delete()
	return nil
}

// TexturePath returns an Option that sets the default texture path.
//
func TexturePath(name string) Option {
	return cfn(func(cfg *config) {
		cfg.texturePath = name
	})
}

func loadTexture(fs ofs.FileSystem, name string, params ...texture.Parameter) (asset, error) {
	r, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return &texImage{src, params}, nil
}

func (m *Manager) PreloadTexture(name string, params ...texture.Parameter) {
	name = path.Join(m.cfg.texturePath, name)
	if !m.loadStart(name) {
		return
	}

	m.cs <- func() {
		a, err := loadTexture(m.fs, name, params...)
		if err != nil {
			m.loadError(name, err)
		} else {
			m.loadComplete(name, a)
		}
	}
}

func (m *Manager) Texture(name string, params ...texture.Parameter) (*texture.Texture, error) {
	name = path.Join(m.cfg.texturePath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		a, err, s := m.assetNoLock(name)
		switch s {
		case stateMissing:
			a, err = m.syncLoadNoLock(name, func(fs ofs.FileSystem, name string) (asset, error) {
				return loadTexture(fs, name, params...)
			})
			params = nil
			if err != nil {
				return nil, err
			}
			fallthrough
		case stateLoaded:
			switch t := a.(type) {
			case *tex:
				tx := (*texture.Texture)(t)
				tx.Parameters(params...)
				return tx, nil
			case *texImage:
				tx := texture.FromImage(t.img, append(t.params, params...)...)
				m.assets[name] = (*tex)(tx)
				return tx, nil
			default:
				return nil, errors.Errorf("asset %s is not a texture", name)
			}
		case stateError:
			return nil, err
		}
		m.cond.Wait()
	}
}

// DiscardTexture removes the named texture from the asset cache along with any associated resources.
//
func (m *Manager) DiscardTexture(name string) error {
	return m.discard(path.Join(m.cfg.texturePath, name))
}
