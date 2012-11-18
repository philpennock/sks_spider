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
	"net"
	"sort"
	"strconv"
	"strings"
)

import (
	btree "github.com/runningwild/go-btree"
)

const hexDigit = "0123456789abcdef"

type CountrySet struct {
	ss btree.SortedSet
}

func NewCountrySet(s string) *CountrySet {
	cs := &CountrySet{ss: btree.NewTree(btreeStringLess)}
	for _, country := range strings.Split(s, ",") {
		cs.ss.Insert(strings.ToUpper(country))
	}
	return cs
}

func (cs *CountrySet) HasCountry(s string) bool {
	return cs.ss.Contains(strings.ToUpper(s))
}

func (cs *CountrySet) String() string {
	cList := make([]string, 0, cs.ss.Len())
	for country := range cs.ss.Data() {
		cList = append(cList, country)
	}
	sort.Strings(cList)
	return strings.Join(cList, ",")
}

func reverseIP(ipstr string) (reversed string, err error) {
	// Crib from net.reverseaddr()
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return "", &net.DNSError{Err: "unrecognized address", Name: ipstr}
	}
	if ip.To4() != nil {
		reversed = strconv.Itoa(int(ip[15])) + "." + strconv.Itoa(int(ip[14])) + "." + strconv.Itoa(int(ip[13])) + "." + strconv.Itoa(int(ip[12]))
		return
	}
	maxLen := len(ip)*4 - 1 // nibble-dot-nibble-dot, no terminating dot here
	buf := make([]byte, 0, maxLen+1)
	// Add it, in reverse, to the buffer
	for i := len(ip) - 1; i >= 0; i-- {
		v := ip[i]
		buf = append(buf, hexDigit[v&0xF])
		buf = append(buf, '.')
		buf = append(buf, hexDigit[v>>4])
		buf = append(buf, '.')
	}
	reversed = string(buf[:maxLen])
	return
}

func CountryForIPString(ipstr string) (country string, err error) {
	rev, err := reverseIP(ipstr)
	if err != nil {
		return "", err
	}
	query := fmt.Sprintf("%s.%s", rev, *flCountriesZone)
	txtList, err := net.LookupTXT(query)
	if err != nil {
		return "", err
	}
	if len(txtList) > 0 {
		return strings.ToUpper(txtList[0]), nil
	}
	return "", fmt.Errorf("No TXT records (and no error) for: %s", query)
}
