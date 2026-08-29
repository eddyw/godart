package envuse

import (
	"os"
	"testing"
)

const envName = "GART_TEST_ENV"

func TestSetenv(t *testing.T) {
	if _, held := os.LookupEnv(envName); held {
		t.Fatalf("%s stands in the environment already", envName)
	}
	t.Setenv(envName, "first")
	if got := os.Getenv(envName); got != "first" {
		t.Errorf("Getenv(%s) = %q, and first is the value", envName, got)
	}
	t.Setenv(envName, "second")
	if got := os.Getenv(envName); got != "second" {
		t.Errorf("Getenv(%s) = %q, and second is the value", envName, got)
	}
}

func TestSetenvCleanup(t *testing.T) {
	if _, held := os.LookupEnv(envName); held {
		t.Errorf("the cleanup of TestSetenv left %s behind", envName)
	}
}

func TestContext(t *testing.T) {
	ctx := t.Context()
	if err := ctx.Err(); err != nil {
		t.Errorf("Err() = %v, and the context of the body is open", err)
	}
	t.Cleanup(func() {
		err := ctx.Err()
		if err == nil {
			t.Error("the context stays open in the cleanup")
			return
		}
		t.Log("the cleanup reads:", err)
	})
}
