package asset

import (
	"io/ioutil"

	"github.com/db47h/ofs"
	"golang.org/x/xerrors"
)

type file []byte

// FilePath returns an Option that sets the default path for raw files.
//
func FilePath(name string) Option {
	return cfn(func(cfg *config) {
		cfg.filePath = name
	})
}

func loadFile(fs ofs.FileSystem, name string) (interface{}, error) {
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

func (m *Manager) File(name string) ([]byte, error) {
	m.m.Lock()
	defer m.m.Unlock()
	for {
		var err error
		a, s := m.getAsset(TypeFile, name)
		switch s {
		case stateMissing:
			a, err = m.syncLoad(TypeFile, name, loadFile)
			if err != nil {
				return nil, err
			}
			fallthrough
		case stateLoaded:
			if data, ok := a.(file); ok {
				return data, nil
			}
			return nil, xerrors.Errorf("asset %s is not a raw file", name)
		}
		m.cond.Wait()
	}
}
