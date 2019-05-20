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

// A Manager manages asynchronous loading and caching of textures, fonts an raw files.
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

func (m *Manager) error(name string, err error) {
	m.m.Lock()
	m.errs[name] = err
	delete(m.ps, name)
	m.cond.Broadcast()
	m.m.Unlock()
}

func (m *Manager) errForAssetNoLock(name string) error {
	if err, ok := m.errs[name]; ok {
		return errors.Wrap(err, name)
	}
	return errors.Wrap(errMissingAsset, name)
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

func (m *Manager) loadStart(name string) (ok bool) {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.ps[name]; ok {
		return false
	}
	if _, ok := m.assets[name]; ok {
		return false
	}
	m.ps[name] = struct{}{}
	return true
}

func (m *Manager) loadComplete(name string, a asset) {
	m.m.Lock()
	delete(m.ps, name)
	m.assets[name] = a
	m.cond.Broadcast()
	m.m.Unlock()
}

func (m *Manager) loadInProgressNoLock(name string) bool {
	_, ok := m.ps[name]
	return ok
}

// QueueSize returns the number of load operations pending and if any error has
// occurred yet.
//
func (m *Manager) QueueSize() (sz int, errors bool) {
	m.m.Lock()
	s := len(m.ps)
	es := len(m.errs)
	m.m.Unlock()
	return s, es > 0
}

// Wait waits until all pending loads complete and returns any load errors that
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

// Discard discards any cached data for the named asset.
//
func (m *Manager) Discard(name string) error {
	m.m.Lock()
	for {
		if a, ok := m.assets[name]; ok {
			delete(m.assets, name)
			m.m.Unlock()
			return a.close()
		}
		if !m.loadInProgressNoLock(name) {
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
