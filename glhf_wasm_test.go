//go:build js && wasm

package glhf

import (
	"strings"
	"testing"
)

func TestPreprocessShaderForES300(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		want    []string // substrings that must be present in output
		notWant []string // substrings that must NOT be present
	}{
		{
			name: "rewrites 330 core to 300 es",
			src:  "#version 330 core\nvoid main() {}\n",
			want: []string{
				"#version 300 es",
				"precision highp float;",
				"precision highp int;",
				"void main() {}",
			},
			notWant: []string{"#version 330 core"},
		},
		{
			name:    "passes 300 es through unchanged",
			src:     "#version 300 es\nprecision mediump float;\nvoid main() {}\n",
			want:    []string{"#version 300 es", "precision mediump float;"},
			notWant: []string{"precision highp float;"},
		},
		{
			name:    "passes 310 es through unchanged",
			src:     "#version 310 es\nvoid main() {}\n",
			want:    []string{"#version 310 es"},
			notWant: []string{"precision highp float;"},
		},
		{
			name: "injects header when no #version directive",
			src:  "void main() {}\n",
			want: []string{
				"#version 300 es",
				"precision highp float;",
				"void main() {}",
			},
		},
		{
			name: "tolerates leading whitespace",
			src:  "\n\n  #version 330 core\nvoid main() {}\n",
			want: []string{"#version 300 es", "void main() {}"},
			notWant: []string{"#version 330 core"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := preprocessShaderForES300(tc.src)
			for _, s := range tc.want {
				if !strings.Contains(got, s) {
					t.Errorf("output missing %q\ngot:\n%s", s, got)
				}
			}
			for _, s := range tc.notWant {
				if strings.Contains(got, s) {
					t.Errorf("output should not contain %q\ngot:\n%s", s, got)
				}
			}
		})
	}
}
