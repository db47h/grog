package assets

import (
	"io/ioutil"
	"path"

	"github.com/db47h/ofs"
	"github.com/pkg/errors"
)

type file []byte

func (file) close() error { return nil }

// FilePath returns an Option that sets the default path for raw files.
//
func FilePath(name string) Option {
	return cfn(func(cfg *config) {
		cfg.filePath = name
	})
}

func loadFile(fs ofs.FileSystem, name string) (asset, error) {
	r, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return file(data), nil
}

func (m *Manager) PreloadFile(name string) {
	name = path.Join(m.cfg.filePath, name)
	if !m.loadStart(name) {
		return
	}
	m.cs <- func() {
		data, err := loadFile(m.fs, name)
		if err != nil {
			m.loadError(name, err)
		} else {
			m.loadComplete(name, data)
		}
	}
}

func (m *Manager) File(name string) ([]byte, error) {
	name = path.Join(m.cfg.filePath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		a, err, s := m.assetNoLock(name)
		switch s {
		case stateMissing:
			a, err = m.syncLoadNoLock(name, loadFile)
			if err != nil {
				return nil, err
			}
			fallthrough
		case stateLoaded:
			if data, ok := a.(file); ok {
				return data, nil
			}
			return nil, errors.Errorf("asset %s is not a raw file", name)
		case stateError:
			return nil, err
		}
		m.cond.Wait()
	}
}

// DiscardFile removes the named file from the asset cache along with any associated resources.
//
func (m *Manager) DiscardFile(name string) error {
	return m.discard(path.Join(m.cfg.filePath, name))
}
