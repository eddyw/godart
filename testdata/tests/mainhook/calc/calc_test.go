package calc

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Println("setup")
	Prefix = "go-"
	code := m.Run()
	fmt.Println("teardown", code)
	os.Exit(code)
}

func TestLabel(t *testing.T) {
	if got := Label("x"); got != "go-x" {
		t.Errorf("Label gives %q, want %q", got, "go-x")
	}
}
