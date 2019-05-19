package assets

import (
	"io/ioutil"
	"path"

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

func (m *Manager) LoadFile(name string) {
	name = path.Join(m.cfg.filePath, name)
	if !m.loadStart(name) {
		return
	}

	m.cs <- func() {
		r, err := m.fs.Open(name)
		if err != nil {
			m.error(name, err)
			return
		}
		data, err := ioutil.ReadAll(r)
		if err != nil {
			m.error(name, err)
			return
		}
		// update
		m.loadComplete(name, file(data))
	}
}

func (m *Manager) File(name string) ([]byte, error) {
	name = path.Join(m.cfg.filePath, name)
	m.m.Lock()
	defer m.m.Unlock()
	for {
		if v, ok := m.assets[name]; ok {
			if data, ok := v.(file); ok {
				return data, nil

			}
			return nil, errors.Errorf("asset %s is not a raw file", name)
		}
		if !m.loadInProgressNoLock(name) {
			// not found. Check if we have any error for this one
			return nil, m.errForAssetNoLock(name)
		}
		m.cond.Wait()
	}
}
