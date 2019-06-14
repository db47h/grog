package asset

import (
	"io"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/xerrors"
)

var errMissingAsset = xerrors.New("asset not found")

// A Manager manages asynchronous (pre)loading and caching of textures, fonts an raw files.
//
type Manager struct {
	fs      FileSystem
	cfg     *config
	m       sync.Mutex
	cond    *sync.Cond
	assets  map[Asset]interface{}
	pending map[Asset]struct{}
}

// NewManager returns a new asset Manager.
//
func NewManager(fs FileSystem, options ...Option) *Manager {
	cfg := new(config)
	for _, o := range options {
		o.set(cfg)
	}

	m := &Manager{
		fs:      fs,
		cfg:     cfg,
		assets:  make(map[Asset]interface{}),
		pending: make(map[Asset]struct{}),
	}
	m.cond = sync.NewCond(&m.m)
	return m
}

type loadState int

const (
	stateMissing = iota
	statePending
	stateLoaded
)

func (m *Manager) lookup(a Asset) (data interface{}, state loadState) {
	if data, ok := m.assets[a]; ok {
		return data, stateLoaded
	}
	if _, ok := m.pending[a]; ok {
		return nil, statePending
	}
	return nil, stateMissing
}

// load loads an asset from disk.
//
func (m *Manager) load(a Asset) (interface{}, error) {
	name := m.cfg.assetPath(a)
	r, err := m.fs.Open(name)
	if err != nil {
		return nil, err
	}
	data, err := loaders[a.Type](r, name)
	if c, ok := r.(io.Closer); ok {
		c.Close()
	}
	return data, err
}

// get returns an asset from cache or synchronously loads it from disk if not
// in the cache. If this asset is being loaded from another goroutine, get will
// wait for the asset to be loaded and return the cached version.
//
func (m *Manager) get(a Asset) (data interface{}, err error) {
	defer func() {
		if err != nil {
			err = xerrors.Errorf("load %s: %w", a, err)
		}
	}()
	for {
		data, s := m.lookup(a)
		switch s {
		case stateMissing:
			m.pending[a] = struct{}{}
			m.m.Unlock()
			data, err := m.load(a)
			m.m.Lock()
			delete(m.pending, a)
			if err != nil {
				return nil, err
			}
			m.assets[a] = data
			return data, nil
		case stateLoaded:
			return data, nil
		}
		m.cond.Wait()
	}
}

// Discard removes the given asset from the cache.
//
func (m *Manager) Discard(a Asset) (err error) {
	defer func() {
		if err != nil {
			err = xerrors.Errorf("discard %s: %w", a, err)
		}
	}()
	m.m.Lock()
	for {
		if aa, ok := m.assets[a]; ok {
			delete(m.assets, a)
			m.m.Unlock()
			if cl, ok := aa.(closer); ok {
				return cl.Close()
			}
			return nil
		}
		if _, ok := m.pending[a]; !ok {
			m.m.Unlock()
			return errMissingAsset
		}
		m.cond.Wait()
	}
}

// Close discards all assets.
//
func (m *Manager) Close() error {
	m.m.Lock()
	var errs errorList
	for _, a := range m.assets {
		if cl, ok := a.(closer); ok {
			if err := cl.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Preload bulk preloads assets. If the flush argument is true, cached assets
// not present in the asset list will be removed from the cache. It returns a
// channel to read preload results from as well as the number of items that will
// actually be preloaded. This item count is informational only and callers
// should rely on the rc channel being closed to ensure that the operation is
// complete.
//
// While preload starts immediately, it will stall after a few assets have been
// preloaded until the rc channel is read from (or Wait is called).
//
// Calling Preload concurrently may result in unexpected side effects, like
// flushing assets that should not be. An alternative is to build the assets
// slice concurrently and have a single goroutine call Preload and Wait.
//
func (m *Manager) Preload(assets []Asset, flush bool) (rc <-chan Result, n int) {
	if flush {
		amap := map[Asset]struct{}{}
		for i := range assets {
			amap[assets[i]] = struct{}{}
		}
		m.m.Lock()
		for k := range m.assets {
			if _, ok := amap[k]; !ok {
				delete(m.assets, k)
			}
		}
	} else {
		m.m.Lock()
	}

	// mark assets as pending and ignore loaded/pending assets
	for i := len(assets) - 1; i > 0; i-- {
		a := &assets[i]
		if a.Type >= typeLast {
			panic(xerrors.Errorf("invalid asset type %d", a.Type))
		}
		_, state := m.lookup(*a)
		if state != stateMissing {
			copy(assets[i:], assets[i+1:])
			assets = assets[:len(assets)-1]
			continue
		}
		m.pending[*a] = struct{}{}
	}
	m.m.Unlock()

	c := make(chan Result)
	go m.preload(assets, c)
	return c, len(assets)
}

func (m *Manager) preload(assets []Asset, rc chan Result) {
	// we use a buffered channel a semaphore to spawn a limited number of
	// workers. This is to prevent excessive simultaneous disk access on
	// mechanical hard drives.
	//
	// goroutines will release the semaphore as soon as they have finished
	// loading the asset but will remain alive until they have sent their result
	// over rc.
	//
	sem := make(chan struct{}, 2*runtime.NumCPU())
	wg := new(sync.WaitGroup)
	for i := range assets {
		sem <- struct{}{}
		wg.Add(1)
		go func(a Asset) {
			data, err := m.load(a)
			m.m.Lock()
			if err != nil {
				err = xerrors.Errorf("preload %s: %w", a, err)
			} else {
				m.assets[a] = data
			}
			delete(m.pending, a)
			m.cond.Broadcast()
			m.m.Unlock()
			<-sem
			rc <- Result{Asset: a, Err: err}
			wg.Done()
		}(assets[i])
	}
	wg.Wait()
	close(rc)
	close(sem)
}

// Wait waits for completion of a previous Preload and returns any load errors.
//
func Wait(rc <-chan Result) error {
	var errs errorList
	for r := range rc {
		if r.Err != nil {
			errs = append(errs, r.Err)
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

type errorList []error

func (e errorList) Error() string {
	var sb strings.Builder
	for i, err := range e {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}
