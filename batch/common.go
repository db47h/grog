package batch

import "github.com/db47h/grog/gl"

const (
	floatsPerVertex = 8
	floatsPerQuad   = floatsPerVertex * 4
	indicesPerQuad  = 6
	batchSize       = 5000
)

func loadShaders() (gl.Program, error) {
	var (
		vertex, frag gl.Shader
		err          error
	)
	vertex, err = gl.NewShader(gl.GL_VERTEX_SHADER, vertexShader)
	if err != nil {
		return 0, err
	}
	defer vertex.Delete()
	frag, err = gl.NewShader(gl.GL_FRAGMENT_SHADER, fragmentShader)
	if err != nil {
		return 0, err
	}
	defer frag.Delete()

	program, err := gl.NewProgram(vertex, frag)
	if err != nil {
		return 0, err
	}

	return program, nil
}

func batchInit(vbo, ebo uint32) {
	indices := make([]uint32, batchSize*indicesPerQuad)
	for i, j := 0, uint32(0); i < len(indices); i, j = i+indicesPerQuad, j+4 {
		indices[i+0] = j + 0
		indices[i+1] = j + 1
		indices[i+2] = j + 2
		indices[i+3] = j + 2
		indices[i+4] = j + 1
		indices[i+5] = j + 3
	}

	gl.BindBuffer(gl.GL_ARRAY_BUFFER, vbo)
	gl.BufferData(gl.GL_ARRAY_BUFFER, batchSize*floatsPerQuad*4, nil, gl.GL_DYNAMIC_DRAW)

	gl.BindBuffer(gl.GL_ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.GL_ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(&indices[0]), gl.GL_STATIC_DRAW)

	gl.Enable(gl.GL_BLEND)
	gl.BlendFunc(gl.GL_ONE, gl.GL_ONE_MINUS_SRC_ALPHA)
}

func batchBegin(vbo, ebo uint32, program gl.Program, pos, color uint32, texture int32) {
	program.Use()
	gl.ActiveTexture(gl.GL_TEXTURE0)
	gl.Uniform1i(texture, 0)
	gl.BindBuffer(gl.GL_ARRAY_BUFFER, vbo)
	gl.BindBuffer(gl.GL_ELEMENT_ARRAY_BUFFER, ebo)
	gl.EnableVertexAttribArray(pos)
	gl.VertexAttribOffset(pos, 4, gl.GL_FLOAT, gl.GL_FALSE, floatsPerVertex*4, 0)
	gl.EnableVertexAttribArray(color)
	gl.VertexAttribOffset(color, 4, gl.GL_FLOAT, gl.GL_FALSE, floatsPerVertex*4, 4*4)
}
