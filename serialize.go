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
