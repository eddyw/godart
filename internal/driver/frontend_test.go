package driver

import (
	"testing"

	compatibility "github.com/eddyw/godart/internal/sdk"
)

func TestFrontendSelection(t *testing.T) {
	versions, err := compatibility.Read()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mode, version, want string
		warn, reject        bool
	}{
		{"auto", "3.13.4", "resident", false, false},
		{"", "3.13.3", "resident", false, false},
		{"source", "3.13.4", "source", false, false},
		{"auto", "3.13.2", "source", true, false},
		{"auto", "3.14.0", "source", true, false},
		{"resident", "3.14.0", "", false, true},
		{"source", "3.12.9", "", false, true},
		{"auto", "4.0.0", "", false, true},
		{"source", "invalid", "", false, true},
		{"unknown", "3.13.4", "", false, true},
	} {
		t.Run(tc.mode+"/"+tc.version, func(t *testing.T) {
			got, warn, err := selectFrontend(tc.mode, tc.version, versions)
			if got != tc.want || warn != tc.warn || (err != nil) != tc.reject {
				t.Fatalf("got %q, warning %v, error %v", got, warn, err)
			}
		})
	}
}
