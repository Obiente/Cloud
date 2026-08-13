package main

import "testing"

func TestParseServiceMode(t *testing.T) {
	tests := map[string]serviceMode{
		"":              {controlPlane: true, proxy: true},
		"all":           {controlPlane: true, proxy: true},
		"control-plane": {controlPlane: true},
		"proxy":         {proxy: true},
		" PROXY ":       {proxy: true},
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := parseServiceMode(input); got != want {
				t.Fatalf("parseServiceMode(%q) = %#v, want %#v", input, got, want)
			}
		})
	}
}
