package asset

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/db47h/grog"
	"golang.org/x/xerrors"
)

type texImage struct {
	img image.Image
}

type tex grog.Texture

func (t *tex) Close() error {
	(*grog.Texture)(t).Delete()
	return nil
}

// TexturePath returns an Option that sets the default texture path.
//
func TexturePath(name string) Option {
	return cfn(func(cfg *config) {
		cfg.texturePath = name
	})
}

func loadTexture(r io.Reader, name string) (interface{}, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return &texImage{src}, nil
}

func (m *Manager) Texture(name string, params ...grog.TextureParameter) (*grog.Texture, error) {
	m.m.Lock()
	defer m.m.Unlock()
	a, err := m.get(Texture(name))
	if err != nil {
		return nil, err
	}
	switch t := a.(type) {
	case *tex:
		tx := (*grog.Texture)(t)
		tx.Parameters(params...)
		return tx, nil
	case *texImage:
		tx := grog.TextureFromImage(t.img, params...)
		m.assets[Asset{TypeTexture, name}] = (*tex)(tx)
		return tx, nil
	default:
		return nil, xerrors.Errorf("asset %s is not a texture", name)
	}
}
