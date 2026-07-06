package main

import (
	"reflect"
	"testing"
)

// TestCoturnURIs exercises coturnURIs across the unset, single, multi, and
// whitespace/empty-entry cases it parses from COTURN_URIS.
func TestCoturnURIs(t *testing.T) {
	tests := []struct {
		name string
		env  string
		set  bool
		want []string
	}{
		{name: "unset", set: false, want: nil},
		{name: "empty", env: "", set: true, want: nil},
		{name: "whitespace only", env: "   ", set: true, want: nil},
		{name: "single", env: "turn:host:3478", set: true, want: []string{"turn:host:3478"}},
		{
			name: "multiple",
			env:  "turn:a:3478,stun:b:3478",
			set:  true,
			want: []string{"turn:a:3478", "stun:b:3478"},
		},
		{
			name: "trims and drops empty entries",
			env:  " turn:a:3478 , , stun:b:3478 ,",
			set:  true,
			want: []string{"turn:a:3478", "stun:b:3478"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("COTURN_URIS", tt.env)
			} else {
				t.Setenv("COTURN_URIS", "")
			}

			got := coturnURIs()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("coturnURIs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestCoturnTTL pins the advertised default TURN credential lifetime.
func TestCoturnTTL(t *testing.T) {
	if coturnTTL.Hours() != 1 {
		t.Errorf("coturnTTL = %v, want 1h", coturnTTL)
	}
}
