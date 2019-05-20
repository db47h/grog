package assets

import (
	"runtime"
	"strings"
	"sync"

	"github.com/db47h/ofs"
	"github.com/pkg/errors"
)

var errMissingAsset = errors.New("asset not found")

type errorList map[string]error

func (e errorList) Error() string {
	var sb strings.Builder
	i := 0
	for k, err := range e {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.Write([]byte{':', ' '})
		sb.WriteString(err.Error())
		i++
	}
	return sb.String()
}

type asset interface {
	close() error
}

// A Manager manages asynchronous (pre)loading and caching of textures, fonts an raw files.
//
type Manager struct {
	fs     ofs.FileSystem
	cfg    *config
	m      sync.Mutex
	cond   *sync.Cond
	errs   errorList
	assets map[string]asset
	ps     map[string]struct{}
	cs     chan func()
}

type config struct {
	texturePath string
	fontPath    string
	filePath    string
}

// Option is implemented by option functions passed as arguments to NewManager.
//
type Option interface {
	set(*config)
}

type cfn func(*config)

func (f cfn) set(cfg *config) {
	f(cfg)
}

// NewManager returns a new asset Manager.
//
func NewManager(fs ofs.FileSystem, options ...Option) *Manager {
	cfg := new(config)
	for _, o := range options {
		o.set(cfg)
	}

	m := &Manager{
		fs:     fs,
		cfg:    cfg,
		errs:   make(errorList),
		assets: make(map[string]asset),
		ps:     make(map[string]struct{}),
		cs:     make(chan func(), 4096),
	}
	m.cond = sync.NewCond(&m.m)
	for i := 0; i < 2*runtime.NumCPU(); i++ {
		go func() {
			for f := range m.cs {
				f()
			}
		}()
	}
	return m
}

func (m *Manager) loadError(name string, err error) {
	m.m.Lock()
	m.errs[name] = errors.Wrap(err, name)
	delete(m.ps, name)
	m.cond.Broadcast()
	m.m.Unlock()
}

// Errors returns load errors. Errors are cleared after each call to this function.
//
func (m *Manager) Errors() error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.errorsNoLock()
}

func (m *Manager) errorsNoLock() error {
	if len(m.errs) == 0 {
		return nil
	}
	es := m.errs
	m.errs = make(errorList)
	return es
}

type loadState int

const (
	stateMissing = iota
	statePending
	stateLoaded
	stateError
)

func (m *Manager) loadStart(name string) bool {
	m.m.Lock()
	_, _, state := m.assetNoLock(name)
	if state == stateMissing {
		m.ps[name] = struct{}{}
	}
	m.m.Unlock()
	return state == stateMissing
}

func (m *Manager) assetNoLock(name string) (a asset, err error, state loadState) {
	if a, ok := m.assets[name]; ok {
		return a, nil, stateLoaded
	}
	if _, ok := m.ps[name]; ok {
		return nil, nil, statePending
	}
	if err := m.errs[name]; err != nil {
		return nil, err, stateError
	}
	return nil, nil, stateMissing
}

func (m *Manager) syncLoadNoLock(name string, f func(fs ofs.FileSystem, name string) (asset, error)) (asset, error) {
	m.ps[name] = struct{}{}
	m.m.Unlock()
	a, err := f(m.fs, name)
	m.m.Lock()
	delete(m.ps, name)
	if err != nil {
		return nil, errors.Wrap(err, name)
	}
	m.assets[name] = a
	return a, nil
}

func (m *Manager) loadComplete(name string, a asset) {
	m.m.Lock()
	delete(m.ps, name)
	m.assets[name] = a
	m.cond.Broadcast()
	m.m.Unlock()
}

// QueueSize returns the number of pending pre-load operations and if any error
// has occurred yet.
//
func (m *Manager) QueueSize() (sz int, errors bool) {
	m.m.Lock()
	s := len(m.ps)
	es := len(m.errs)
	m.m.Unlock()
	return s, es > 0
}

// Wait waits until all pending pre-loads complete and returns any errors that
// occurred so far. Errors are cleared after each call to this function.
//
func (m *Manager) Wait() error {
	m.m.Lock()
	defer m.m.Unlock()
	for len(m.ps) > 0 {
		m.cond.Wait()
	}
	return m.errorsNoLock()
}

func (m *Manager) discard(name string) error {
	m.m.Lock()
	for {
		if a, ok := m.assets[name]; ok {
			delete(m.assets, name)
			m.m.Unlock()
			return a.close()
		}
		if err, ok := m.errs[name]; ok {
			delete(m.errs, name)
			return err
		}
		if _, ok := m.ps[name]; !ok {
			m.m.Unlock()
			return errors.Wrap(errMissingAsset, name)
		}
		m.cond.Wait()
	}
}

// Close discards all assets and stops any spawned goroutines. Any subsequent
// call to a Load function will cause a panic.
//
func (m *Manager) Close() error {
	close(m.cs)
	_ = m.Wait()
	errs := make(errorList)
	for name, a := range m.assets {
		if err := a.close(); err != nil {
			errs[name] = err
		}
	}
	if len(errs) > 0 {
		return errs
	}
	m.assets = nil
	m.errs = nil
	return nil
}
