package asset

import (
	"io"
	"io/ioutil"

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

func loadFile(r io.Reader, name string) (interface{}, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return file(data), nil
}

func (m *Manager) File(name string) ([]byte, error) {
	m.m.Lock()
	defer m.m.Unlock()
	a, err := m.load(File(name))
	if err != nil {
		return nil, err
	}
	if data, ok := a.(file); ok {
		return data, nil
	}
	return nil, xerrors.Errorf("asset %s is not a raw file", name)
}
