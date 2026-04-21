//go:build !js

package glhf

import "github.com/go-gl/gl/v3.3-core/gl"

// Init initializes OpenGL by loading function pointers from the active OpenGL context.
// This function must be manually run inside the main thread (using "github.com/gopxl/mainthread/v2"
// package).
//
// It must be called under the presence of an active OpenGL context, e.g., always after calling
// window.MakeContextCurrent(). Also, always call this function when switching contexts.
func Init() {
	err := gl.Init()
	if err != nil {
		panic(err)
	}
	gl.Enable(gl.BLEND)
	gl.Enable(gl.SCISSOR_TEST)
	gl.BlendEquation(gl.FUNC_ADD)
}

// Clear clears the current framebuffer or window with the given color.
func Clear(r, g, b, a float32) {
	gl.ClearColor(r, g, b, a)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// Bounds sets the drawing bounds in pixels. Drawing outside bounds is always discarted.
//
// Calling this function is equivalent to setting viewport and scissor in OpenGL.
func Bounds(x, y, w, h int) {
	gl.Viewport(int32(x), int32(y), int32(w), int32(h))
	gl.Scissor(int32(x), int32(y), int32(w), int32(h))
}

// BlendFactor represents a source or destination blend factor.
type BlendFactor int

// Here's the list of all blend factors.
const (
	One              = BlendFactor(gl.ONE)
	Zero             = BlendFactor(gl.ZERO)
	SrcAlpha         = BlendFactor(gl.SRC_ALPHA)
	DstAlpha         = BlendFactor(gl.DST_ALPHA)
	OneMinusSrcAlpha = BlendFactor(gl.ONE_MINUS_SRC_ALPHA)
	OneMinusDstAlpha = BlendFactor(gl.ONE_MINUS_DST_ALPHA)
	SrcColor         = BlendFactor(gl.SRC_COLOR)
	DstColor         = BlendFactor(gl.DST_COLOR)
	OneMinusSrcColor = BlendFactor(gl.ONE_MINUS_SRC_COLOR)
	OneMinusDstColor = BlendFactor(gl.ONE_MINUS_DST_COLOR)
)

// BlendFunc sets the source and destination blend factor.
func BlendFunc(src, dst BlendFactor) {
	gl.BlendFunc(uint32(src), uint32(dst))
}

// BlendFuncSeparate sets separate blend factors for RGB and alpha channels.
func BlendFuncSeparate(srcRGB, dstRGB, srcAlpha, dstAlpha BlendFactor) {
	gl.BlendFuncSeparate(uint32(srcRGB), uint32(dstRGB), uint32(srcAlpha), uint32(dstAlpha))
}

// BlendEquationMode identifies a blend equation.
type BlendEquationMode int

// Supported blend equations.
const (
	FuncAdd             = BlendEquationMode(gl.FUNC_ADD)
	FuncSubtract        = BlendEquationMode(gl.FUNC_SUBTRACT)
	FuncReverseSubtract = BlendEquationMode(gl.FUNC_REVERSE_SUBTRACT)
)

// BlendEquation sets the active blend equation.
func BlendEquation(mode BlendEquationMode) {
	gl.BlendEquation(uint32(mode))
}

// ActiveTexture selects the active texture unit (0-based index). Unit N
// corresponds to GL_TEXTURE0 + N.
func ActiveTexture(unit int) {
	gl.ActiveTexture(uint32(gl.TEXTURE0 + int32(unit)))
}
