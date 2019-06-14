package asset

import (
	"io"
	"path"
)

var loaders = [...]func(io.Reader, string) (interface{}, error){
	TypeFont:    loadFont,
	TypeTexture: loadTexture,
	TypeFile:    loadFile,
}

type config struct {
	texturePath string
	fontPath    string
	filePath    string
}

func (c *config) assetPath(a Asset) string {
	switch a.Type {
	case TypeFont:
		return path.Join(c.fontPath, a.Name)
	case TypeTexture:
		return path.Join(c.texturePath, a.Name)
	case TypeFile:
		return path.Join(c.filePath, a.Name)
	}
	panic("invalid type")
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

// closer is implemented by assets that need to free up resources.
//
type closer interface {
	Close() error
}

// A FileSystem implements access to named resources. Names are file paths using
// '/' as a path separator.
//
type FileSystem interface {
	Open(filename string) (io.Reader, error)
}

// Type designates the type of an asset.
//
type Type int

const (
	TypeFont = iota
	TypeTexture
	TypeFile
	typeLast
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
