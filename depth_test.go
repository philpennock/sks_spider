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
	"strings"
	"testing"
)

const TEST_DATA_FILE = "data/hostdump-20121117.json"

func TestDepthSnapshot(t *testing.T) {
	hostmap, err := LoadJSONFromFile(TEST_DATA_FILE)
	if err != nil {
		t.Fatalf("Failed to load \"%s\": %s", TEST_DATA_FILE, err)
	}

	depthSorted := GenerateDepthSorted(hostmap)
	distance := -1
	tld := ""
	for _, hostname := range depthSorted {
		newDistance := hostmap[hostname].Distance
		switch {
		case newDistance < distance:
			t.Fatalf("Not depth-sorted; host \"%s\" depth %d, were at depth %d", hostname, newDistance, distance)
		case newDistance == distance:
			sections := strings.Split(hostname, ".")
			newTld := sections[len(sections)-1]
			if newTld < tld {
				t.Fatalf("Within depth %d TLD went backwards, ... '%s', '%s' ...", distance, tld, newTld)
			}
			tld = newTld
		default:
			distance = newDistance
			tld = ""
		}
	}
	if distance < 1 {
		t.Fatalf("No distance achieved; distance == %d", distance)
	}
	t.Logf("Depth OK; %d entries, max distance %d", len(depthSorted), distance)
}
