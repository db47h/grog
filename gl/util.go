//go:generate gogl -gl 2.1 -v -p gl

package gl

/*
#include "gl.h"
#include <stdlib.h>

static void vertexAttribOffset(GLuint index, GLint size, GLenum type_, GLboolean normalized, GLsizei stride, GLintptr offset) {
	glVertexAttribPointer(index, size, type_, normalized, stride, (void *)offset);
}

static const char *newShader(GLuint *shaderPtr, GLenum shaderType, const char *src, GLint len) {
	char *err = NULL;
	GLuint shader = glCreateShader(shaderType);
	glShaderSource(shader, 1, &src, &len);
	glCompileShader(shader);

	int success;
	glGetShaderiv(shader, GL_COMPILE_STATUS, &success);
	if (!success) {
		int loglen = 0;
		glGetShaderiv(shader, GL_INFO_LOG_LENGTH, &loglen);
		err = malloc(loglen);
		glGetShaderInfoLog(shader, loglen, NULL, err);
		glDeleteShader(shader);
		shader = 0;
	}
	*shaderPtr = shader;
	return err;
}

static const char *newProgram(GLuint *pPtr, GLuint *shaders, GLint len) {
	char *err = NULL;
	GLuint p = glCreateProgram();

	while (len-- > 0) glAttachShader(p, *shaders++);

	glLinkProgram(p);

	int success;
	glGetProgramiv(p, GL_LINK_STATUS, &success);
	if (!success) {
		int loglen = 0;
		glGetProgramiv(p, GL_INFO_LOG_LENGTH, &loglen);
		err = malloc(loglen);
		glGetProgramInfoLog(p, loglen, NULL, err);
		glDeleteProgram(p);
		p = 0;
	}
	*pPtr = p;
	return err;
}

static GLint getAttribLocation(GLuint program, const char *name) {
	GLint rv = glGetAttribLocation(program, name);
	free((void *)name);
	return rv;
}

static GLint getUniformLocation(GLuint program, const char *name) {
	GLint rv = glGetUniformLocation(program, name);
	free((void *)name);
	return rv;
}

static void batchBegin(GLuint program, GLuint vbo, GLuint ebo, GLuint aPos, GLuint aColor, GLint uTexture) {
	glUseProgram(program);

	glActiveTexture(GL_TEXTURE0);
	glUniform1i(uTexture, 0);

	glBindBuffer(GL_ARRAY_BUFFER, vbo);
	glBindBuffer(GL_ELEMENT_ARRAY_BUFFER, ebo);
	glEnableVertexAttribArray(aPos);
	glVertexAttribPointer(aPos, 4, GL_FLOAT, GL_FALSE, 8*sizeof(GLfloat), 0);
	glEnableVertexAttribArray(aColor);
	glVertexAttribPointer(aColor, 4, GL_FLOAT, GL_FALSE, 8*sizeof(GLfloat), (void *)(4*sizeof(GLfloat)));
}

static void batchDraw(GLuint tex, GLfloat *vertices, GLsizeiptr size) {
	glBindTexture(GL_TEXTURE_2D, tex);
	glBufferSubData(GL_ARRAY_BUFFER, 0, size*sizeof(GLfloat), vertices);
	glDrawElements(GL_TRIANGLES, (size / 32)*6, GL_UNSIGNED_INT, NULL);
}

*/
import "C"

import (
	"image/color"
	"unsafe"

	"github.com/pkg/errors"
)

// GetGoString is a wrapper around GetString that returns a Go string.
//
func GetGoString(name uint32) string {
	return C.GoString((*C.char)(unsafe.Pointer(GetString(name))))
}

// VertexAttribOffset is a variant of VertexAttribPointer for cases where pointer is an offset and not a real pointer.
//
func VertexAttribOffset(index uint32, size int32, type_ uint32, normalized byte, stride int32, offset int) {
	C.vertexAttribOffset(C.GLuint(index), C.GLint(size), C.GLenum(type_), C.GLboolean(normalized), C.GLsizei(stride), C.GLintptr(offset))
}

func Sizeof(v interface{}) int {
	switch v := v.(type) {
	case []int8:
		return len(v)
	case []uint8:
		return len(v)
	case []int16:
		return len(v) * 2
	case []uint16:
		return len(v) * 2
	case []int32:
		return len(v) * 4
	case []uint32:
		return len(v) * 4
	case []int64:
		return len(v) * 8
	case []uint64:
		return len(v) * 8
	case []float32:
		return len(v) * 4
	case []float64:
		return len(v) * 8
	case int8:
		return 1
	case uint8:
		return 1
	case int16:
		return 2
	case uint16:
		return 2
	case int32:
		return 4
	case uint32:
		return 4
	case int64:
		return 8
	case uint64:
		return 8
	case float32:
		return 4
	case float64:
		return 8
	default:
		panic(errors.Errorf("sizeof: invalid type %T", v))
	}
}

func Ptr(v interface{}) unsafe.Pointer {
	if v == nil {
		return unsafe.Pointer(nil)
	}
	switch v := v.(type) {
	case *int:
		return unsafe.Pointer(v)
	case *uint:
		return unsafe.Pointer(v)
	case *int8:
		return unsafe.Pointer(v)
	case *uint8:
		return unsafe.Pointer(v)
	case *int16:
		return unsafe.Pointer(v)
	case *uint16:
		return unsafe.Pointer(v)
	case *int32:
		return unsafe.Pointer(v)
	case *uint32:
		return unsafe.Pointer(v)
	case *int64:
		return unsafe.Pointer(v)
	case *uint64:
		return unsafe.Pointer(v)
	case *float32:
		return unsafe.Pointer(v)
	case *float64:
		return unsafe.Pointer(v)
	}
	panic("cannot convert")
}

type Shader uint32

func NewShader(typ uint32, source []byte) (Shader, error) {
	var s uint32
	err := C.newShader((*C.GLuint)(&s), C.GLenum(typ), (*C.char)(unsafe.Pointer(&source[0])), C.GLint(len(source)))
	if err != nil {
		errMsg := C.GoString(err)
		C.free(unsafe.Pointer(err))
		return 0, errors.New(errMsg)
	}
	return Shader(s), nil
}

func (s Shader) Delete() {
	DeleteShader(uint32(s))
}

type Program uint32

func NewProgram(shaders ...Shader) (Program, error) {
	var p Program

	err := C.newProgram((*C.GLuint)(&p), (*C.GLuint)(unsafe.Pointer(&shaders[0])), C.GLint(len(shaders)))
	if err != nil {
		errMsg := C.GoString(err)
		C.free(unsafe.Pointer(err))
		return 0, errors.New(errMsg)
	}
	return p, nil
}

func (p Program) Delete() {
	DeleteProgram(uint32(p))
}

func (p Program) Use() {
	UseProgram(uint32(p))
}

func (p Program) AttribLocation(name string) (uint32, error) {
	r := int32(C.getAttribLocation(C.GLuint(p), C.CString(name)))
	if r < 0 {
		return ^uint32(0), errors.Errorf("unknown attribute %s", name)
	}
	return uint32(r), nil
}

func (p Program) UniformLocation(name string) int32 {
	return int32(C.getUniformLocation(C.GLuint(p), C.CString(name)))
}

// func BatchBegin(program Program, vbo uint32, ebo uint32, aPos uint32, aColor uint32, uTexture int32) {
// 	C.batchBegin(C.GLuint(program), C.GLuint(vbo), C.GLuint(ebo), C.GLuint(aPos), C.GLuint(aColor), C.GLint(uTexture))
// }

// func BatchDraw(tex uint32, vertices []float32) {
// 	C.batchDraw(C.GLuint(tex), (*C.GLfloat)(&vertices[0]), C.GLsizeiptr(len(vertices)));
// }

// Color implements color.Color. It stores alpha premultiplied color components in
// the range [0, 1],
//
type Color struct {
	R, G, B, A float32
}

// RGBA implements color.Color.
//
func (c Color) RGBA() (r, g, b, a uint32) {
	return uint32(r*0xffff) & 0xffff, uint32(g*0xffff) & 0xffff, uint32(b*0xffff) & 0xffff, uint32(a*0xffff) & 0xffff
}

// ColorModel converts any color.Color to a GLColor; i.e. the result can safely be
// casted to a GLColor.
//
var ColorModel = color.ModelFunc(colorModel)

func colorModel(c color.Color) color.Color {
	if _, ok := c.(Color); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	return Color{R: float32(r) / 0xffff, G: float32(g) / 0xffff, B: float32(b) / 0xffff, A: float32(a) / 0xffff}
}
