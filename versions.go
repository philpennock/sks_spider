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
	"fmt"
	"regexp"
	"strconv"
)

type SksVersion struct {
	Major, Minor, Release uint
	Tag                   string
}

var sksVersionRegexp *regexp.Regexp

func init() {
	sksVersionRegexp = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(\+?)$`)
}

func NewSksVersion(s string) *SksVersion {
	matches := sksVersionRegexp.FindStringSubmatch(s)
	if matches == nil {
		return nil
	}
	v1, err := strconv.ParseUint(matches[1], 10, 0)
	if err != nil {
		return nil
	}
	v2, err := strconv.ParseUint(matches[2], 10, 0)
	if err != nil {
		return nil
	}
	v3, err := strconv.ParseUint(matches[3], 10, 0)
	if err != nil {
		return nil
	}
	return &SksVersion{Major: uint(v1), Minor: uint(v2), Release: uint(v3), Tag: matches[4]}
}

func (sv *SksVersion) String() string {
	return fmt.Sprintf("%d.%d.%d%s", sv.Major, sv.Minor, sv.Release, sv.Tag)
}

func (sv *SksVersion) IsAtLeast(min *SksVersion) bool {
	if sv.Major < min.Major {
		return false
	} else if sv.Major > min.Major {
		return true
	}
	if sv.Minor < min.Minor {
		return false
	} else if sv.Minor > min.Minor {
		return true
	}
	if sv.Release < min.Release {
		return false
	} else if sv.Release > min.Release {
		return true
	}
	if len(min.Tag) > 0 && len(sv.Tag) == 0 {
		return false
	}
	return true
}
