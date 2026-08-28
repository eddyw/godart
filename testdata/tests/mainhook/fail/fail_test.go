package fail

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	fmt.Println("status", code)
	os.Exit(code)
}

func TestDouble(t *testing.T) {
	if got := Double(2); got != 4 {
		t.Errorf("Double gives %d, want 4", got)
	}
}
