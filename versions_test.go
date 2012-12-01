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
	"testing"
)

func checkedNewSksVersion(t *testing.T, version string) *SksVersion {
	sv := NewSksVersion(version)
	if sv == nil {
		t.Fatalf("Failed to parse version from \"%s\"", version)
	}
	return sv
}

func TestVersionMatching(t *testing.T) {
	validVersions := [...]string{
		"1.1.4", "1.1.4+",
		"0.0.0", "10000.1000.10000",
	}
	invalidVersions := [...]string{
		"", "+", "-1.0.0", "1000000000000000000000000000000000.2.3",
		"1.2.3++", "1.2.3 ",
	}
	for _, ver := range validVersions {
		checkedNewSksVersion(t, ver)
	}
	for _, ver := range invalidVersions {
		sv := NewSksVersion(ver)
		if sv != nil {
			t.Fatalf("Unexpectedly succeeded parsing version from \"%s\": %s", ver, sv)
		}
	}
	min1 := checkedNewSksVersion(t, "2.4.6")
	min2 := checkedNewSksVersion(t, "2.4.6+")
	for _, v := range []*SksVersion{min1, min2} {
		if !v.IsAtLeast(v) {
			t.Fatalf("Version not at least itself: %s", v)
		}
	}
	if !min2.IsAtLeast(min1) {
		t.Fatalf("Plus variant not at least non-plus, wanted %s >= %s", min2, min1)
	}
	if min1.IsAtLeast(min2) {
		t.Fatalf("Version %s apparently >= %s (shouldn't be)", min1, min2)
	}
	older := [...]string{
		"2.4.5", "2.3.10", "1.5.10",
	}
	newer := [...]string{
		"3.1.2", "2.5.0", "2.4.7",
	}
	for _, v := range older {
		sv := checkedNewSksVersion(t, v)
		if sv.IsAtLeast(min1) {
			t.Fatalf("Version %s apparently >= %s (shouldn't be)", sv, min1)
		}
	}
	for _, v := range newer {
		sv := checkedNewSksVersion(t, v)
		if !sv.IsAtLeast(min1) {
			t.Fatalf("Version %s not >= %s (should be)", sv, min1)
		}
	}
}
