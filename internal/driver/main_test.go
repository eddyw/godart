package driver_test

import (
	"flag"
	"os"
	"testing"
)

var update = flag.Bool("update", false, "write the golden files again")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
