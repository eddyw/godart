package driver_test

import (
	"maps"
	"slices"
	"testing"
)

func TestShardCoverage(t *testing.T) {
	oldIndex, oldCount := *shardIndex, *shardCount
	defer func() { *shardIndex, *shardCount = oldIndex, oldCount }()
	names := []string{"new", "slow", "fast", "another", "last"}
	for _, count := range []int{1, 3, 8} {
		*shardCount = count
		seen := make(map[string]int)
		for i := range count {
			*shardIndex = i
			selection := testShard(t, names)
			reversed := slices.Clone(names)
			slices.Reverse(reversed)
			other := testShard(t, reversed)
			if !maps.Equal(selection, other) {
				t.Fatal("source enumeration order changed the shard")
			}
			for name := range selection {
				seen[name]++
			}
		}
		for _, name := range names {
			if seen[name] != 1 {
				t.Fatalf("%s ran %d times across %d shards", name, seen[name], count)
			}
		}
	}
}
