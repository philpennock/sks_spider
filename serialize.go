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
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func (hostmap HostMap) DumpJSONToFile(filename string) error {
	fh, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = hostmap.DumpJSON(fh)
	if err != nil {
		fh.Close()
		return err
	}
	err = fh.Close()
	return err
}

func (hostmap HostMap) DumpJSON(out io.Writer) error {
	var b []byte
	var err error

	fmt.Fprintf(out, "{\n")
	need_comma := false
	for name, node := range hostmap {
		if node == nil {
			continue
		}
		if need_comma {
			fmt.Fprintf(out, ",\n")
		}
		b, err = json.Marshal(node)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "\"%s\":%s", name, b)
		need_comma = true
	}
	fmt.Fprintf(out, "\n}\n")
	return nil
}

func LoadJSONFromFile(filename string) (HostMap, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	hostmap := make(map[string]*SksNode)
	decoder := json.NewDecoder(fh)
	err = decoder.Decode(&hostmap)
	if err != nil {
		return nil, err
	}

	for n := range hostmap {
		if hostmap[n] != nil {
			hostmap[n].initialised = true
		}
	}
	return hostmap, nil
}
