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
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
)

func GetMembershipHosts() ([]string, error) {
	fh, err := os.Open(*flSksMembershipFile)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	reader := bufio.NewReader(fh)
	hosts := make([]string, 0, 100)
	matcher := regexp.MustCompile(`^([A-Za-z0-9]\S+)\s+\d`)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		if line == "" && err == io.EOF {
			break
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matches := matcher.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		hosts = append(hosts, matches[1])
	}
	return hosts, nil
}

func GetMembershipAsNodemap() (map[string]*SksNode, error) {
	hosts, err := GetMembershipHosts()
	if err != nil {
		return nil, err
	}
	nodes := make(map[string]*SksNode, len(hosts))
	for _, h := range hosts {
		nodes[h] = nil
	}
	return nodes, nil
}
