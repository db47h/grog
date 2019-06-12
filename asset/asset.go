package asset

import (
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/db47h/ofs"
	"golang.org/x/xerrors"
)

var errMissingAsset = xerrors.New("asset not found")

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

type Closer interface {
	Close() error
}

// A Manager manages asynchronous (pre)loading and caching of textures, fonts an raw files.
//
type Manager struct {
	fs      ofs.FileSystem
	cfg     *config
	m       sync.Mutex
	cond    *sync.Cond
	assets  map[Asset]interface{}
	pending map[Asset]struct{}
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

func (m *Manager) lookup(t Type, name string) (a interface{}, state loadState) {
	k := Asset{t, name}
	if a, ok := m.assets[k]; ok {
		return a, stateLoaded
	}
	if _, ok := m.pending[k]; ok {
		return nil, statePending
	}
	return nil, stateMissing
}

// load returns an asset from cache or synchronously loads it from disk if not
// in the cache. If this asset is being loaded from another goroutine, load will
// wait for the asset to be loaded and return the cached version.
//
func (m *Manager) load(t Type, name string, f func(fs ofs.FileSystem, name string) (interface{}, error)) (interface{}, error) {
	for {
		var err error
		a, s := m.lookup(t, name)
		switch s {
		case stateMissing:
			k := Asset{t, name}
			m.pending[k] = struct{}{}
			m.m.Unlock()
			a, err = f(m.fs, m.assetPath(&k))
			m.m.Lock()
			delete(m.pending, k)
			if err != nil {
				return nil, xerrors.Errorf("load %s: %w", Asset{t, name}, err)
			}
			m.assets[k] = a
			return a, nil
		case stateLoaded:
			return a, nil
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
			if cl, ok := aa.(Closer); ok {
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
		if cl, ok := a.(Closer); ok {
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

// Type designates the type of an asset.
//
type Type int

const (
	TypeFont = iota
	TypeTexture
	TypeFile
)

// Asset uniquely describes an asset.
//
type Asset struct {
	Type
	Name string
}

func (a Asset) String() string {
	switch a.Type {
	case TypeFont:
		return "font asset " + a.Name
	case TypeTexture:
		return "texture asset " + a.Name
	case TypeFile:
		return "file asset " + a.Name
	}
	return "unknown asset " + a.Name
}

// Result wraps the result from preloading an asset.
//
type Result struct {
	Asset
	Err error
}

func Font(name string) Asset    { return Asset{TypeFont, name} }
func Texture(name string) Asset { return Asset{TypeTexture, name} }
func File(name string) Asset    { return Asset{TypeFile, name} }

func (m *Manager) assetPath(a *Asset) string {
	switch a.Type {
	case TypeFont:
		return path.Join(m.cfg.fontPath, a.Name)
	case TypeTexture:
		return path.Join(m.cfg.texturePath, a.Name)
	case TypeFile:
		return path.Join(m.cfg.filePath, a.Name)
	}
	return a.Name
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
		_, state := m.lookup(a.Type, a.Name)
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
	// spawn a limited number of workers. This is to prevent excessive
	// simultaneous disk access on mechanical hard drives.
	c := make(chan func())
	for i := 0; i < 2*runtime.NumCPU(); i++ {
		go func(c chan func()) {
			for f := range c {
				f()
			}
		}(c)
	}

	// we buffer resultBufSize results, forward to rc then close all channels
	// when all results have been posted.
	const resultBufSize = 1024
	rcTemp := make(chan Result, resultBufSize)
	go func(n int) {
		for i := 0; i < n; i++ {
			rc <- (<-rcTemp)
		}
		close(rc)
		close(rcTemp)
	}(len(assets))

	for i := range assets {
		a := assets[i]
		c <- func() {
			var (
				data interface{}
				err  error
				name = m.assetPath(&a)
			)
			switch a.Type {
			case TypeFont:
				data, err = loadFont(m.fs, name)
			case TypeTexture:
				data, err = loadTexture(m.fs, name)
			case TypeFile:
				data, err = loadFile(m.fs, name)
			default:
				panic("Unknown asset type")
			}
			m.m.Lock()
			if err != nil {
				err = xerrors.Errorf("preload %s: %w", a, err)
			} else {
				m.assets[a] = data
			}
			delete(m.pending, a)
			m.cond.Broadcast()
			m.m.Unlock()
			rcTemp <- Result{Asset: a, Err: err}
		}
	}
	close(c)
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
