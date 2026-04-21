// Package glhf provides abstractions around the basic OpenGL primitives and operations.
//
// On desktop targets all calls should be done from the main thread using the
// "github.com/gopxl/mainthread/v2" package.
//
// A parallel WebGL2 backend is built under GOOS=js GOARCH=wasm. The exported
// API is the same, except that the caller must install a WebGL2 rendering
// context once via SetContext before invoking Init or any constructor.
//
// This package deliberately does not handle nor report trivial OpenGL errors, it's up to you to
// cause none. It does of course report errors like shader compilation error and such.
package glhf
