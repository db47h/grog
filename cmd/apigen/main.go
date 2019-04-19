//go:generate curl -L --compressed -o gl.xml https://raw.githubusercontent.com/KhronosGroup/OpenGL-Registry/master/xml/gl.xml

package main

import (
	"encoding/xml"
	"flag"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var (
	api     string
	version Version
	profile string
	tags    string
)

func main() {
	var (
		out string
		in  string
	)

	flag.StringVar(&api, "api", "gl", "api to generate: gl, gles1 or gles2")
	flag.Var(&version, "v", "api version (default 1.0 for gles1, 2.0 for gles2, 3.1 for gl)")
	flag.StringVar(&profile, "p", "", "default profile")
	flag.StringVar(&out, "o", "gl_generated", "base `name` of output file(s)")
	flag.StringVar(&in, "i", "gl.xml", "input `filename`")
	flag.StringVar(&tags, "t", "", "build `tags` for the generated file")

	flag.Parse()

	switch api {
	case "gles1", "gles2", "gl":
	default:
		panic("invalid api")
	}

	if version.Major == 0 && version.Minor == 0 {
		switch api {
		case "gles1":
			version.Set("1.0")
		case "gles2":
			version.Set("2.0")
		case "gl":
			version.Set("3.3")
		}
	}

	x, err := os.Open(in)
	if err != nil {
		panic(err)
	}
	r, err := decodeRegistry(x)
	x.Close()
	if err != nil {
		panic(err)
	}

	fname := "gl.tmpl"
	t, err := template.New(fname).ParseFiles("templates/" + fname)
	if err != nil {
		panic(err)
	}
	err = t.Execute(os.Stdout, r)
	if err != nil {
		panic(err)
	}
}

type Command struct {
	Type   Type
	Name   string
	Params []Param
}

type Param struct {
	Type Type
	Name string
}

func (c *Command) GoName() string {
	if strings.HasPrefix(c.Name, "gl") {
		return strings.ToUpper(c.Name[2:3]) + c.Name[3:]
	}
	return c.Name
}

func (c *Command) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var xc struct {
		Proto struct {
			Type string `xml:"ptype"`
			Ptr  string `xml:",chardata"`
			Name string `xml:"name"`
		} `xml:"proto"`
		Params []struct {
			Type string `xml:"ptype"`
			Ptr  string `xml:",chardata"`
			Name string `xml:"name"`
		} `xml:"param"`
	}
	err := d.DecodeElement(&xc, &start)
	if err != nil {
		return err
	}

	c.Type = MkType(xc.Proto.Type, xc.Proto.Ptr)
	c.Name = xc.Proto.Name
	c.Params = make([]Param, 0, len(xc.Params))
	for _, xp := range xc.Params {
		if xp.Type == "" {
			xp.Type = "void"
		}
		c.Params = append(c.Params, Param{
			Name: paramName(xp.Name),
			Type: MkType(xp.Type, xp.Ptr),
		})
	}
	return nil
}

var reserved = map[string]struct{}{
	"int":        {},
	"int8":       {},
	"int16":      {},
	"int32":      {},
	"uint32":     {},
	"uint":       {},
	"uint8":      {},
	"uint16":     {},
	"int64":      {},
	"uint64":     {},
	"uintptr":    {},
	"float32":    {},
	"float64":    {},
	"complex64":  {},
	"complex128": {},
	"string":     {},
	"byte":       {},
	"rune":       {},
	"bool":       {},

	"break":       {},
	"default":     {},
	"func":        {},
	"interface":   {},
	"select":      {},
	"case":        {},
	"defer":       {},
	"go":          {},
	"map":         {},
	"struct":      {},
	"chan":        {},
	"else":        {},
	"goto":        {},
	"package":     {},
	"switch":      {},
	"const":       {},
	"fallthrough": {},
	"if":          {},
	"range":       {},
	"type":        {},
	"continue":    {},
	"for":         {},
	"import":      {},
	"return":      {},
	"var":         {},
}

func paramName(n string) string {
	if _, ok := reserved[n]; ok {
		return n + "_"
	}
	return n
}

type Version struct {
	Major int
	Minor int
}

func (v *Version) String() string {
	return strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Major)
}

func (v *Version) Get() interface{} {
	return *v
}

func (v *Version) Set(s string) error {
	end := strings.IndexRune(s, '.')
	if end < 0 {
		end = len(s)
	}

	n, err := strconv.ParseInt(s[:end], 10, 32)
	if err != nil {
		return err
	}
	v.Major = int(n)
	if end >= len(s) {
		v.Minor = 0
		return nil
	}
	n, err = strconv.ParseInt(s[end+1:], 10, 32)
	if err != nil {
		return err
	}
	v.Minor = int(n)
	return nil
}

func (v *Version) Less(rhs *Version) bool {
	if v.Major == rhs.Major {
		return v.Minor < rhs.Minor
	}
	return v.Major < rhs.Major
}

func decodeRegistry(r io.Reader) (*Registry, error) {
	var reg registry
	d := xml.NewDecoder(r)
	err := d.Decode(&reg)
	if err != nil {
		return nil, err
	}

	return &Registry{
		API:      api,
		Tags:     tags,
		Enums:    sortEnums(reg.Enums),
		Commands: sortCommands(reg.Commands),
	}, nil
}

type Registry struct {
	API      string
	Tags     string
	Enums    []Enum
	Commands []*Command
}

type Enum struct {
	Name  string
	Value string
}

func (e *Enum) GoName() string {
	if strings.HasPrefix(e.Name, "GL_") {
		return e.Name[3:]
	}
	return e.Name
}

func sortEnums(em map[string]string) []Enum {
	enums := make([]Enum, 0, len(em))
	for k, v := range em {
		enums = append(enums, Enum{k, v})
	}
	sort.Slice(enums, func(i, j int) bool { return enums[i].Name < enums[j].Name })
	return enums
}

func sortCommands(cm map[string]*Command) []*Command {
	cmds := make([]*Command, 0, len(cm))
	for _, v := range cm {
		cmds = append(cmds, v)
	}
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })
	return cmds
}
