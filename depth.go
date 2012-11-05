package sks_spider

import (
	"sort"
)

func GenerateDepthSorted(hostmap HostMap) []string {
	var by_depth = make(map[int][]string, 7)
	var per_depth_len = len(hostmap)
	var ordered_entries = make([]string, 0, len(hostmap))

	for name, node := range hostmap {
		depth := node.Distance
		if _, ok := by_depth[depth]; !ok {
			by_depth[depth] = make([]string, 0, per_depth_len)
		}
		by_depth[depth] = append(by_depth[depth], name)
	}
	if _, ok := by_depth[-1]; ok {
		shuffle := by_depth[-1]
		delete(by_depth, -1)
		max := 0
		for d := range by_depth {
			if d > max {
				max = d
			}
		}
		by_depth[max+1] = shuffle
	}
	available_depths := make([]int, 0, len(by_depth))
	for d := range by_depth {
		available_depths = append(available_depths, d)
	}
	sort.Ints(available_depths)

	for _, d := range available_depths {
		subrange := by_depth[d]
		HostSort(subrange)
		ordered_entries = append(ordered_entries, subrange...)
	}

	return ordered_entries
}
