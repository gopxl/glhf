//go:build js && wasm

// WebGL2 backend for glhf. Mirrors the exported surface of the desktop OpenGL 3.3
// backend (shader.go, texture.go, frame.go, vertex.go, orphan.go, util.go) and
// dispatches to a WebGL2RenderingContext via syscall/js.

package glhf

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/go-gl/mathgl/mgl32"
)

// --- Shader ----------------------------------------------------------------

type Shader struct {
	program    js.Value
	vertexFmt  AttrFormat
	uniformFmt AttrFormat
	uniformLoc []js.Value
	id         uint32
}

var nextID uint32

func newID() uint32 {
	nextID++
	return nextID
}

// preprocessShaderForES300 adapts GLSL 330 core source (written for the
// desktop backend) to WebGL2's required GLSL ES 3.00. If the source already
// declares an ES version, it is returned untouched. Otherwise:
//   - the `#version ...` line is rewritten to `#version 300 es`
//   - `precision highp float; precision highp int;` is injected on the next
//     line so fragment shaders get mandatory precision qualifiers.
func preprocessShaderForES300(src string) string {
	trimmed := strings.TrimLeft(src, " \t\r\n")
	if strings.Contains(trimmed, "#version 300 es") ||
		strings.Contains(trimmed, "#version 310 es") {
		return src
	}

	// Locate (and drop) the first #version line, if any.
	lines := strings.SplitN(trimmed, "\n", 2)
	rest := trimmed
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#version") {
		if len(lines) == 2 {
			rest = lines[1]
		} else {
			rest = ""
		}
	}
	return "#version 300 es\nprecision highp float;\nprecision highp int;\n" + rest
}

func compileShader(kind int, src string) (js.Value, error) {
	src = preprocessShaderForES300(src)
	sh := gl.Call("createShader", kind)
	gl.Call("shaderSource", sh, src)
	gl.Call("compileShader", sh)
	if !gl.Call("getShaderParameter", sh, glEnum("COMPILE_STATUS")).Bool() {
		log := gl.Call("getShaderInfoLog", sh).String()
		gl.Call("deleteShader", sh)
		kindName := "vertex"
		if kind == glEnum("FRAGMENT_SHADER") {
			kindName = "fragment"
		}
		return js.Null(), fmt.Errorf("error compiling %s shader: %s", kindName, log)
	}
	return sh, nil
}

func NewShader(vertexFmt, uniformFmt AttrFormat, vertexShader, fragmentShader string) (*Shader, error) {
	if !glReady {
		return nil, fmt.Errorf("glhf: WebGL context not set; call glhf.SetContext first")
	}

	vs, err := compileShader(glEnum("VERTEX_SHADER"), vertexShader)
	if err != nil {
		return nil, err
	}
	fs, err := compileShader(glEnum("FRAGMENT_SHADER"), fragmentShader)
	if err != nil {
		gl.Call("deleteShader", vs)
		return nil, err
	}

	prog := gl.Call("createProgram")
	gl.Call("attachShader", prog, vs)
	gl.Call("attachShader", prog, fs)
	gl.Call("linkProgram", prog)
	gl.Call("deleteShader", vs)
	gl.Call("deleteShader", fs)
	if !gl.Call("getProgramParameter", prog, glEnum("LINK_STATUS")).Bool() {
		log := gl.Call("getProgramInfoLog", prog).String()
		gl.Call("deleteProgram", prog)
		return nil, fmt.Errorf("error linking shader program: %s", log)
	}

	s := &Shader{
		program:    prog,
		vertexFmt:  vertexFmt,
		uniformFmt: uniformFmt,
		uniformLoc: make([]js.Value, len(uniformFmt)),
		id:         newID(),
	}
	for i, u := range uniformFmt {
		s.uniformLoc[i] = gl.Call("getUniformLocation", prog, u.Name)
	}
	return s, nil
}

func (s *Shader) ID() uint32                { return s.id }
func (s *Shader) VertexFormat() AttrFormat  { return s.vertexFmt }
func (s *Shader) UniformFormat() AttrFormat { return s.uniformFmt }
func (s *Shader) Begin()                    { gl.Call("useProgram", s.program) }
func (s *Shader) End()                      { gl.Call("useProgram", js.Null()) }

func (s *Shader) SetUniformAttr(uniform int, value interface{}) (ok bool) {
	loc := s.uniformLoc[uniform]
	if loc.IsNull() || !loc.Truthy() {
		return false
	}
	switch s.uniformFmt[uniform].Type {
	case Int:
		gl.Call("uniform1i", loc, int(value.(int32)))
	case Float:
		gl.Call("uniform1f", loc, float64(value.(float32)))
	case Vec2:
		v := value.(mgl32.Vec2)
		gl.Call("uniform2fv", loc, float32ToJS(v[:]))
	case Vec3:
		v := value.(mgl32.Vec3)
		gl.Call("uniform3fv", loc, float32ToJS(v[:]))
	case Vec4:
		v := value.(mgl32.Vec4)
		gl.Call("uniform4fv", loc, float32ToJS(v[:]))
	case Mat2:
		v := value.(mgl32.Mat2)
		gl.Call("uniformMatrix2fv", loc, false, float32ToJS(v[:]))
	case Mat23:
		v := value.(mgl32.Mat2x3)
		gl.Call("uniformMatrix2x3fv", loc, false, float32ToJS(v[:]))
	case Mat24:
		v := value.(mgl32.Mat2x4)
		gl.Call("uniformMatrix2x4fv", loc, false, float32ToJS(v[:]))
	case Mat3:
		v := value.(mgl32.Mat3)
		gl.Call("uniformMatrix3fv", loc, false, float32ToJS(v[:]))
	case Mat32:
		v := value.(mgl32.Mat3x2)
		gl.Call("uniformMatrix3x2fv", loc, false, float32ToJS(v[:]))
	case Mat34:
		v := value.(mgl32.Mat3x4)
		gl.Call("uniformMatrix3x4fv", loc, false, float32ToJS(v[:]))
	case Mat4:
		v := value.(mgl32.Mat4)
		gl.Call("uniformMatrix4fv", loc, false, float32ToJS(v[:]))
	case Mat42:
		v := value.(mgl32.Mat4x2)
		gl.Call("uniformMatrix4x2fv", loc, false, float32ToJS(v[:]))
	case Mat43:
		v := value.(mgl32.Mat4x3)
		gl.Call("uniformMatrix4x3fv", loc, false, float32ToJS(v[:]))
	default:
		panic("set uniform attr: invalid attribute type")
	}
	return true
}

// --- Texture ---------------------------------------------------------------

type Texture struct {
	handle        js.Value
	width, height int
	smooth        bool
	id            uint32
}

func NewTexture(width, height int, smooth bool, pixels []uint8) *Texture {
	t := &Texture{
		handle: gl.Call("createTexture"),
		width:  width,
		height: height,
		id:     newID(),
	}
	prev := gl.Call("getParameter", glEnum("TEXTURE_BINDING_2D"))
	gl.Call("bindTexture", glEnum("TEXTURE_2D"), t.handle)

	var data js.Value
	if len(pixels) == width*height*4 {
		data = uint8ToJS(pixels)
	} else {
		data = uint8ToJS(make([]byte, width*height*4))
	}
	gl.Call("texImage2D",
		glEnum("TEXTURE_2D"), 0, glEnum("RGBA"),
		width, height, 0,
		glEnum("RGBA"), glEnum("UNSIGNED_BYTE"),
		data,
	)

	// WebGL2 does not support CLAMP_TO_BORDER; fall back to CLAMP_TO_EDGE.
	gl.Call("texParameteri", glEnum("TEXTURE_2D"), glEnum("TEXTURE_WRAP_S"), glEnum("CLAMP_TO_EDGE"))
	gl.Call("texParameteri", glEnum("TEXTURE_2D"), glEnum("TEXTURE_WRAP_T"), glEnum("CLAMP_TO_EDGE"))

	t.SetSmooth(smooth)

	gl.Call("bindTexture", glEnum("TEXTURE_2D"), prev)
	return t
}

func (t *Texture) ID() uint32    { return t.id }
func (t *Texture) Width() int    { return t.width }
func (t *Texture) Height() int   { return t.height }
func (t *Texture) Smooth() bool  { return t.smooth }
func (t *Texture) Handle() js.Value { return t.handle }

func (t *Texture) SetSmooth(smooth bool) {
	t.smooth = smooth
	filter := glEnum("NEAREST")
	if smooth {
		filter = glEnum("LINEAR")
	}
	gl.Call("texParameteri", glEnum("TEXTURE_2D"), glEnum("TEXTURE_MIN_FILTER"), filter)
	gl.Call("texParameteri", glEnum("TEXTURE_2D"), glEnum("TEXTURE_MAG_FILTER"), filter)
}

func (t *Texture) Begin() {
	gl.Call("bindTexture", glEnum("TEXTURE_2D"), t.handle)
}

func (t *Texture) End() {
	gl.Call("bindTexture", glEnum("TEXTURE_2D"), js.Null())
}

func (t *Texture) SetPixels(x, y, w, h int, pixels []uint8) {
	if len(pixels) != w*h*4 {
		panic("set pixels: wrong number of pixels")
	}
	gl.Call("texSubImage2D",
		glEnum("TEXTURE_2D"), 0,
		x, y, w, h,
		glEnum("RGBA"), glEnum("UNSIGNED_BYTE"),
		uint8ToJS(pixels),
	)
}

// Pixels reads a sub-rectangle from the texture. WebGL has no glGetTexImage, so
// this attaches the texture to a transient framebuffer and uses readPixels.
func (t *Texture) Pixels(x, y, w, h int) []uint8 {
	fbo := gl.Call("createFramebuffer")
	prev := gl.Call("getParameter", glEnum("FRAMEBUFFER_BINDING"))
	gl.Call("bindFramebuffer", glEnum("FRAMEBUFFER"), fbo)
	gl.Call("framebufferTexture2D",
		glEnum("FRAMEBUFFER"), glEnum("COLOR_ATTACHMENT0"),
		glEnum("TEXTURE_2D"), t.handle, 0,
	)

	buf := uint8ToJS(make([]byte, w*h*4))
	gl.Call("readPixels", x, y, w, h, glEnum("RGBA"), glEnum("UNSIGNED_BYTE"), buf)

	out := make([]byte, w*h*4)
	js.CopyBytesToGo(out, buf)

	gl.Call("bindFramebuffer", glEnum("FRAMEBUFFER"), prev)
	gl.Call("deleteFramebuffer", fbo)
	return out
}

// --- Frame -----------------------------------------------------------------

type Frame struct {
	fb  js.Value
	tex *Texture
	id  uint32
}

func NewFrame(width, height int, smooth bool) *Frame {
	f := &Frame{
		fb:  gl.Call("createFramebuffer"),
		tex: NewTexture(width, height, smooth, nil),
		id:  newID(),
	}
	prev := gl.Call("getParameter", glEnum("FRAMEBUFFER_BINDING"))
	gl.Call("bindFramebuffer", glEnum("FRAMEBUFFER"), f.fb)
	gl.Call("framebufferTexture2D",
		glEnum("FRAMEBUFFER"), glEnum("COLOR_ATTACHMENT0"),
		glEnum("TEXTURE_2D"), f.tex.handle, 0,
	)
	gl.Call("bindFramebuffer", glEnum("FRAMEBUFFER"), prev)
	return f
}

func (f *Frame) ID() uint32        { return f.id }
func (f *Frame) Texture() *Texture { return f.tex }
func (f *Frame) Begin()            { gl.Call("bindFramebuffer", glEnum("FRAMEBUFFER"), f.fb) }
func (f *Frame) End()              { gl.Call("bindFramebuffer", glEnum("FRAMEBUFFER"), js.Null()) }

// Blit copies a rectangle from this Frame to dst (or default framebuffer if nil).
// Uses WebGL2 blitFramebuffer.
func (f *Frame) Blit(dst *Frame, sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int) {
	var dstFB js.Value
	if dst != nil {
		dstFB = dst.fb
	} else {
		dstFB = js.Null()
	}

	prevRead := gl.Call("getParameter", glEnum("READ_FRAMEBUFFER_BINDING"))
	prevDraw := gl.Call("getParameter", glEnum("DRAW_FRAMEBUFFER_BINDING"))
	gl.Call("bindFramebuffer", glEnum("READ_FRAMEBUFFER"), f.fb)
	gl.Call("bindFramebuffer", glEnum("DRAW_FRAMEBUFFER"), dstFB)

	filter := glEnum("NEAREST")
	if f.tex.smooth {
		filter = glEnum("LINEAR")
	}

	gl.Call("blitFramebuffer",
		sx0, sy0, sx1, sy1,
		dx0, dy0, dx1, dy1,
		glEnum("COLOR_BUFFER_BIT"), filter,
	)

	gl.Call("bindFramebuffer", glEnum("READ_FRAMEBUFFER"), prevRead)
	gl.Call("bindFramebuffer", glEnum("DRAW_FRAMEBUFFER"), prevDraw)
}

// --- VertexSlice -----------------------------------------------------------

type VertexSlice struct {
	va   *vertexArray
	i, j int
}

type vertexArray struct {
	vao    js.Value
	vbo    js.Value
	cap    int
	format AttrFormat
	stride int
	offset []int
	shader *Shader
	// shadow mirrors the GPU buffer so VertexData() can return without a GPU
	// readback. getBufferSubData on a DYNAMIC_DRAW buffer causes pipeline
	// stalls (WebGL warns); keeping the data CPU-side sidesteps that.
	shadow []float32
}

const vertexArrayMinCap = 4

func MakeVertexSlice(shader *Shader, length, capacity int) *VertexSlice {
	if length > capacity {
		panic("failed to make vertex slice: len > cap")
	}
	if capacity < vertexArrayMinCap {
		capacity = vertexArrayMinCap
	}
	va := newVertexArray(shader, capacity)
	return &VertexSlice{va: va, i: 0, j: length}
}

func newVertexArray(shader *Shader, capacity int) *vertexArray {
	stride := shader.VertexFormat().Size()
	va := &vertexArray{
		vao:    gl.Call("createVertexArray"),
		vbo:    gl.Call("createBuffer"),
		cap:    capacity,
		format: shader.VertexFormat(),
		stride: stride,
		offset: make([]int, len(shader.VertexFormat())),
		shader: shader,
		shadow: make([]float32, capacity*(stride/4)),
	}
	off := 0
	for i, attr := range va.format {
		switch attr.Type {
		case Float, Vec2, Vec3, Vec4:
		default:
			panic("failed to create vertex array: invalid attribute type")
		}
		va.offset[i] = off
		off += attr.Type.Size()
	}

	prevVAO := gl.Call("getParameter", glEnum("VERTEX_ARRAY_BINDING"))
	prevVBO := gl.Call("getParameter", glEnum("ARRAY_BUFFER_BINDING"))
	gl.Call("bindVertexArray", va.vao)
	gl.Call("bindBuffer", glEnum("ARRAY_BUFFER"), va.vbo)

	gl.Call("bufferData", glEnum("ARRAY_BUFFER"),
		capacity*stride, glEnum("DYNAMIC_DRAW"))

	for i, attr := range va.format {
		loc := gl.Call("getAttribLocation", shader.program, attr.Name).Int()
		if loc < 0 {
			continue
		}
		var size int
		switch attr.Type {
		case Float:
			size = 1
		case Vec2:
			size = 2
		case Vec3:
			size = 3
		case Vec4:
			size = 4
		}
		gl.Call("vertexAttribPointer", loc, size, glEnum("FLOAT"), false, stride, va.offset[i])
		gl.Call("enableVertexAttribArray", loc)
	}

	gl.Call("bindBuffer", glEnum("ARRAY_BUFFER"), prevVBO)
	gl.Call("bindVertexArray", prevVAO)
	return va
}

func (vs *VertexSlice) VertexFormat() AttrFormat { return vs.va.format }
func (vs *VertexSlice) Stride() int              { return vs.va.stride / 4 }
func (vs *VertexSlice) Len() int                 { return vs.j - vs.i }
func (vs *VertexSlice) Cap() int                 { return vs.va.cap - vs.i }

func (vs *VertexSlice) SetLen(length int) {
	vs.End()
	*vs = vs.grow(length)
	vs.Begin()
}

func (vs VertexSlice) grow(length int) VertexSlice {
	if length <= vs.Cap() {
		return VertexSlice{va: vs.va, i: vs.i, j: vs.i + length}
	}
	newCap := vs.Cap()
	if newCap < 1024 {
		newCap += newCap
	} else {
		newCap += newCap / 4
	}
	if newCap < length {
		newCap = length
	}
	newVs := VertexSlice{
		va: newVertexArray(vs.va.shader, newCap),
		i:  0,
		j:  length,
	}
	newVs.Begin()
	newVs.Slice(0, vs.Len()).SetVertexData(vs.VertexData())
	newVs.End()
	return newVs
}

func (vs *VertexSlice) Slice(i, j int) *VertexSlice {
	if i < 0 || j < i || j > vs.va.cap {
		panic("failed to slice vertex slice: index out of range")
	}
	return &VertexSlice{va: vs.va, i: vs.i + i, j: vs.i + j}
}

func (vs *VertexSlice) SetVertexData(data []float32) {
	if len(data)/vs.Stride() != vs.Len() {
		panic("set vertex data: wrong length of vertices")
	}
	if len(data) == 0 {
		return
	}
	copy(vs.va.shadow[vs.i*vs.Stride():], data)
	gl.Call("bufferSubData", glEnum("ARRAY_BUFFER"),
		vs.i*vs.va.stride, float32ToJS(data))
}

func (vs *VertexSlice) VertexData() []float32 {
	n := (vs.j - vs.i) * vs.Stride()
	if n == 0 {
		return nil
	}
	out := make([]float32, n)
	copy(out, vs.va.shadow[vs.i*vs.Stride():vs.j*vs.Stride()])
	return out
}

func (vs *VertexSlice) Draw() {
	gl.Call("drawArrays", glEnum("TRIANGLES"), vs.i, vs.j-vs.i)
}

func (vs *VertexSlice) Begin() {
	gl.Call("bindVertexArray", vs.va.vao)
	gl.Call("bindBuffer", glEnum("ARRAY_BUFFER"), vs.va.vbo)
}

func (vs *VertexSlice) End() {
	gl.Call("bindBuffer", glEnum("ARRAY_BUFFER"), js.Null())
	gl.Call("bindVertexArray", js.Null())
}

// --- Global GL state -------------------------------------------------------

func Init() {
	if !glReady {
		panic("glhf: WebGL context not set; call glhf.SetContext first")
	}
	gl.Call("enable", glEnum("BLEND"))
	gl.Call("enable", glEnum("SCISSOR_TEST"))
	gl.Call("blendEquation", glEnum("FUNC_ADD"))
}

func Clear(r, g, b, a float32) {
	gl.Call("clearColor", float64(r), float64(g), float64(b), float64(a))
	gl.Call("clear", glEnum("COLOR_BUFFER_BIT"))
}

func Bounds(x, y, w, h int) {
	gl.Call("viewport", x, y, w, h)
	gl.Call("scissor", x, y, w, h)
}

func BlendFunc(src, dst BlendFactor) {
	gl.Call("blendFunc", int(src), int(dst))
}

func BlendFuncSeparate(srcRGB, dstRGB, srcAlpha, dstAlpha BlendFactor) {
	gl.Call("blendFuncSeparate", int(srcRGB), int(dstRGB), int(srcAlpha), int(dstAlpha))
}

type BlendEquationMode int

const (
	FuncAdd             = BlendEquationMode(0x8006)
	FuncSubtract        = BlendEquationMode(0x800A)
	FuncReverseSubtract = BlendEquationMode(0x800B)
)

func BlendEquation(mode BlendEquationMode) {
	gl.Call("blendEquation", int(mode))
}

// ActiveTexture selects the active texture unit (0-based). Unit N maps to
// WebGL's TEXTURE0 + N (0x84C0 + N).
func ActiveTexture(unit int) {
	gl.Call("activeTexture", 0x84C0+unit)
}

type BlendFactor int

// WebGL2 numeric enum values (match desktop OpenGL).
const (
	One              = BlendFactor(1)
	Zero             = BlendFactor(0)
	SrcAlpha         = BlendFactor(0x0302)
	DstAlpha         = BlendFactor(0x0304)
	OneMinusSrcAlpha = BlendFactor(0x0303)
	OneMinusDstAlpha = BlendFactor(0x0305)
	SrcColor         = BlendFactor(0x0300)
	DstColor         = BlendFactor(0x0306)
	OneMinusSrcColor = BlendFactor(0x0301)
	OneMinusDstColor = BlendFactor(0x0307)
)
