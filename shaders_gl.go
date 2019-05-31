// +build !gles2

package grog

var vertexShader = []byte(`#version 130
attribute vec4 aPos;
attribute vec4 aColor;

varying vec4 vTexColor;
varying vec2 vTexCoords;

uniform mat4 uProjection;

void main()
{
	gl_Position = uProjection * vec4(aPos.xy, 0.0, 1.0);
    vTexColor = aColor;
    vTexCoords = aPos.zw;
}
`)

var fragmentShader = []byte(`#version 130
precision mediump float;
  
varying vec4 vTexColor;
varying vec2 vTexCoords;

uniform sampler2D uTexture;

void main()
{
    gl_FragColor = vTexColor * texture2D(uTexture, vTexCoords);
}
`)
