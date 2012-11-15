/*
   Copyright 2009-2012 Phil Pennock

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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
