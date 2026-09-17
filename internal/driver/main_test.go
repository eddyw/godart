package driver_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
)

var update = flag.Bool("update", false, "write the golden files again")
var shardIndex = flag.Int("shard-index", 0, "zero-based shard of the large test suites")
var shardCount = flag.Int("shard-count", 1, "number of shards of the large test suites")

var caseDurations = sync.OnceValues(func() (map[string]float64, error) {
	data, err := os.ReadFile("testdata/ci-durations.json")
	if err != nil {
		return nil, err
	}
	var durations map[string]float64
	err = json.Unmarshal(data, &durations)
	return durations, err
})

// Longest cases go first, into the shard with the least estimated work.
// New cases receive a nonzero estimate and remain part of the selection.
func testShard(t *testing.T, names []string) map[string]bool {
	t.Helper()
	durations, err := caseDurations()
	if err != nil {
		t.Fatal(err)
	}
	weight := func(name string) float64 {
		key := t.Name() + "/" + name
		if testing.Short() && t.Name() == "TestTestsOfGoroot" {
			key = t.Name() + "/short/" + name
		}
		if d := durations[key]; d > 0 {
			return d
		}
		return 1
	}
	names = slices.Clone(names)
	slices.SortFunc(names, func(a, b string) int {
		if weight(a) > weight(b) {
			return -1
		}
		if weight(a) < weight(b) {
			return 1
		}
		return strings.Compare(a, b)
	})
	loads := make([]float64, *shardCount)
	selected := make(map[string]bool)
	for _, name := range names {
		shard := 0
		for i := 1; i < len(loads); i++ {
			if loads[i] < loads[shard] {
				shard = i
			}
		}
		loads[shard] += weight(name)
		if shard == *shardIndex {
			selected[name] = true
		}
	}
	return selected
}

func TestMain(m *testing.M) {
	flag.Parse()
	if *shardCount < 1 || *shardIndex < 0 || *shardIndex >= *shardCount {
		fmt.Fprintln(os.Stderr, "require shard-count >= 1 and 0 <= shard-index < shard-count")
		os.Exit(2)
	}
	os.Exit(m.Run())
}
