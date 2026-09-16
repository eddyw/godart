package driver_test

import (
	"flag"
	"fmt"
	"hash/fnv"
	"os"
	"testing"
)

var update = flag.Bool("update", false, "write the golden files again")
var shardIndex = flag.Int("shard-index", 0, "zero-based shard of the program, Go language, and GOROOT suites")
var shardCount = flag.Int("shard-count", 1, "number of shards of the program, Go language, and GOROOT suites")

func inTestShard(name string) bool {
	h := fnv.New64a()
	h.Write([]byte(name))
	return int(h.Sum64()%uint64(*shardCount)) == *shardIndex
}

func TestMain(m *testing.M) {
	flag.Parse()
	if *shardCount < 1 || *shardIndex < 0 || *shardIndex >= *shardCount {
		fmt.Fprintln(os.Stderr, "require shard-count >= 1 and 0 <= shard-index < shard-count")
		os.Exit(2)
	}
	os.Exit(m.Run())
}
