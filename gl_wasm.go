//go:build js && wasm

package glhf

import (
	"syscall/js"
	"unsafe"
)

// Package-level WebGL2 state. Callers (pixel backend, standalone consumers) must
// invoke SetContext exactly once, after creating a WebGL2RenderingContext, before
// any glhf.Init / New* / Make* call runs.

var (
	gl             js.Value
	glReady        bool
	uint8ArrayCtor js.Value
	f32ArrayCtor   js.Value
)

// SetContext installs the WebGL2 rendering context used by all glhf calls.
func SetContext(ctx js.Value) {
	gl = ctx
	uint8ArrayCtor = js.Global().Get("Uint8Array")
	f32ArrayCtor = js.Global().Get("Float32Array")
	glReady = true
}

// Context returns the installed WebGL2 context (empty js.Value if unset).
func Context() js.Value { return gl }

// glEnum looks up a WebGL constant by name (e.g. "TEXTURE_2D").
func glEnum(name string) int { return gl.Get(name).Int() }

// uint8ToJS copies a Go byte slice into a new JS Uint8Array.
func uint8ToJS(b []byte) js.Value {
	u8 := uint8ArrayCtor.New(len(b))
	if len(b) > 0 {
		js.CopyBytesToJS(u8, b)
	}
	return u8
}

// float32ToJS copies a Go float32 slice into a new JS Float32Array.
func float32ToJS(f []float32) js.Value {
	if len(f) == 0 {
		return f32ArrayCtor.New(0)
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(&f[0])), len(f)*4)
	u8 := uint8ToJS(b)
	return f32ArrayCtor.New(u8.Get("buffer"), u8.Get("byteOffset"), len(f))
}

// float32FromJS copies a JS Float32Array back into a Go float32 slice.
func float32FromJS(arr js.Value, n int) []float32 {
	out := make([]float32, n)
	if n == 0 {
		return out
	}
	// Read bytes via a Uint8Array view over the same buffer.
	u8 := uint8ArrayCtor.New(arr.Get("buffer"), arr.Get("byteOffset"), n*4)
	b := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*4)
	js.CopyBytesToGo(b, u8)
	return out
}
