// Package glhf provides abstractions around the basic OpenGL primitives and operations.
//
// All calls should be done from the main thread using "github.com/gopxl/mainthread/v2" package.
//
// This package deliberately does not handle nor report trivial OpenGL errors, it's up to you to
// cause none. It does of course report errors like shader compilation error and such.
package glhf
