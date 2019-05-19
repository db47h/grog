package assets

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"path"

	"github.com/db47h/grog/texture"
	"github.com/pkg/errors"
)

type texImage struct {
	img    image.Image
	params []texture.Parameter
}

func (*texImage) close() error { return nil }

type tex texture.Texture

func (t *tex) close() error { (*texture.Texture)(t).Delete(); return nil }

// TexturePath returns an Option that sets the default texture path.
//
func TexturePath(name string) Option {
	return cfn(func(cfg *config) {
		cfg.texturePath = name
	})
}

func (m *Manager) LoadTexture(name string, params ...texture.Parameter) {
	name = path.Join(m.cfg.texturePath, name)
	if !m.loadStart(name) {
		return
	}

	m.cs <- func() {
		r, err := m.fs.Open(name)
		if err != nil {
			m.error(name, err)
			return
		}
		src, _, err := image.Decode(r)
		if err != nil {
			m.error(name, err)
			return
		}
		// update
		m.loadComplete(name, &texImage{src, params})
	}
}

func (m *Manager) Texture(name string) (*texture.Texture, error) {
	name = path.Join(m.cfg.texturePath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		if t, ok := m.assets[name]; ok {
			switch t := t.(type) {
			case *tex:
				return (*texture.Texture)(t), nil
			case *texImage:
				tx := texture.FromImage(t.img, t.params...)
				m.assets[name] = (*tex)(tx)
				return tx, nil
			default:
				return nil, errors.Errorf("asset %s is not a texture", name)
			}
		}
		if !m.loadInProgressNoLock(name) {
			// not found. Check if we have any error for this one
			return nil, m.errForAssetNoLock(name)
		}
		m.cond.Wait()
	}
}
