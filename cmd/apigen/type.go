package main

import (
	"fmt"
	"strings"
)

var types = map[string][]string{
	"void":                 {"", "unsafe.Pointer", "*unsafe.Pointer"},
	"GLboolean":            {"byte", "*byte", "**byte"},
	"GLbyte":               {"int8", "*int8", "**int8"},
	"GLubyte":              {"uint8", "*uint8", "**uint8"},
	"GLchar":               {"int8", "*int8", "**int8"},
	"GLuchar":              {"byte", "*byte", "**byte"},
	"GLcharARB":            {"int8", "*int8", "**byte"},
	"GLshort":              {"int16", "*int16", "**int16"},
	"GLushort":             {"uint16", "*uint16", "**uint16"},
	"GLint":                {"int32", "*int32", "**int32"},
	"GLuint":               {"uint32", "*uint32", "**uint32"},
	"GLfixed":              {"int32", "*int32", "**int32"},
	"GLint64":              {"int64", "*int64", "**int64"},
	"GLint64EXT":           {"int64", "*int64", "**int64"},
	"GLuint64":             {"uint64", "*uint64", "**uint64"},
	"GLuint64EXT":          {"uint64", "*uint64", "**uint64"},
	"GLsizei":              {"int32", "*int32", "**int32"},
	"GLenum":               {"uint32", "*uint32", "**uint32"},
	"GLintptr":             {"int", "*int", "**int"},
	"GLintptrARB":          {"int", "*int", "**int"},
	"GLvdpauSurfaceNV":     {"int", "*int", "**int"},
	"GLsizeiptr":           {"int", "*int", "**int"},
	"GLsizeiptrARB":        {"int", "*int", "**int"},
	"GLbitfield":           {"int32", "*int32", "**int32"},
	"GLhalf":               {"uint16", "*uint16", "**uint16"},
	"GLhalfNV":             {"uint16", "*uint16", "**uint16"},
	"GLfloat":              {"float32", "*float32", "**float32"},
	"GLclampf":             {"float32", "*float32", "**float32"},
	"GLclampx":             {"int32", "*int32", "**int32"},
	"GLdouble":             {"float64", "*float64", "**float64"},
	"GLclampd":             {"float64", "*float64", "**float64"},
	"GLhandleARB":          {"uintptr", "uintptr", "uintptr"},
	"GLsync":               {"unsafe.Pointer", "unsafe.Pointer", "unsafe.Pointer"},
	"GLeglClientBufferEXT": {"unsafe.Pointer", "unsafe.Pointer", "unsafe.Pointer"},
	"GLeglImageOES":        {"unsafe.Pointer", "unsafe.Pointer", "unsafe.Pointer"},
	"GLDEBUGPROC":          {"unsafe.Pointer", "unsafe.Pointer", "unsafe.Pointer"},
	"GLDEBUGPROCARB":       {"unsafe.Pointer", "unsafe.Pointer", "unsafe.Pointer"},
	"GLDEBUGPROCAMD":       {"unsafe.Pointer", "unsafe.Pointer", "unsafe.Pointer"},
	"GLDEBUGPROCKHR":       {"unsafe.Pointer", "unsafe.Pointer", "unsafe.Pointer"},
	"GLVULKANPROCNV":       {"unsafe.Pointer", "unsafe.Pointer", "unsafe.Pointer"},
	"struct _cl_context":   {"", "unsafe.Pointer", "unsafe.Pointer"},
	"struct _cl_event":     {"", "unsafe.Pointer", "unsafe.Pointer"},
}

type Type struct {
	Name  string
	Ptr   int
	Const bool
}

func MkType(name string, raw string) Type {
	if name == "" {
		name = "void"
	}
	if _, ok := types[name]; !ok {
		panic(fmt.Errorf("invalid type %s", name))
	}
	return Type{
		Name:  name,
		Ptr:   strings.Count(raw, "*"),
		Const: strings.Index(raw, "const") >= 0,
	}
}

func (t *Type) CName() string {
	var sb strings.Builder
	if t.Const {
		sb.WriteString("const ")
	}
	sb.WriteString(t.Name)
	if t.Ptr > 0 {
		sb.WriteString(" *")
		if t.Ptr > 1 {
			if t.Const {
				sb.WriteString("const*")
			} else {
				sb.WriteByte('*')
			}
		}
	}
	return sb.String()
}

func (t *Type) GoName(ret bool) string {
	if ret && t.Name == "void" {
		return ""
	}
	return types[t.Name][t.Ptr]
}

func (t *Type) ToC(arg string) string {
	gn := t.GoName(false)
	switch gn {
	case "string":
		return "C.CString(" + arg + ")"
	case "unsafe.Pointer", "*unsafe.Pointer":
		if t.Name == "void" {
			return arg
		}
		fallthrough
	default:
		if t.Ptr > 1 {
			return "(" + strings.Repeat("*", t.Ptr) + "C." + t.Name + ")(unsafe.Pointer(" + arg + "))"
		}
		if t.Ptr > 0 {
			return "(" + strings.Repeat("*", t.Ptr) + "C." + t.Name + ")(" + arg + ")"
		}
		return "C." + t.Name + "(" + arg + ")"
	}
}

func (t *Type) ToGo() string {
	gn := types[t.Name][t.Ptr]
	if gn == "" {
		panic(fmt.Errorf("cannot convert C type %s to a Go type", t.Name))
	}
	if strings.IndexByte(gn, '*') >= 0 {
		return "(" + gn + ")"
	}
	return gn
}
