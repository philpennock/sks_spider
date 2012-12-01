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

func TestAddressSkip(t *testing.T) {
	shouldBeRejected := [...]string{
		"0.1.2.3",
		"127.0.0.2",
		"169.254.0.0",
		"169.254.2.4",
		"169.254.255.255",
		"172.16.0.0",
		"172.16.0.255",
		"172.31.255.255",
		"192.0.2.42",
		"241.2.3.4",
		"2001:db8::1",
	}
	shouldBeAllowed := [...]string{
		"172.32.0.0",
		"2001:1db8::1",
	}
	for _, wantFail := range shouldBeRejected {
		if !IPDisallowed(wantFail) {
			t.Fatalf("IP [%s] was cleared for use, should have been rejected", wantFail)
		}
	}
	for _, wantAllow := range shouldBeAllowed {
		if IPDisallowed(wantAllow) {
			t.Fatalf("IP [%s] was rejected, should be clear for use", wantAllow)
		}
	}
}
