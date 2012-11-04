package sks_spider

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func (spider *Spider) DumpJSONToFile(filename string) error {
	fh, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = spider.DumpJSON(fh)
	if err != nil {
		fh.Close()
		return err
	}
	err = fh.Close()
	return err
}

func (spider *Spider) DumpJSON(out io.Writer) error {
	var b []byte
	var err error

	fmt.Fprintf(out, "{\n")
	need_comma := false
	for name, node := range spider.serverInfos {
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

func LoadJSONFromFile(filename string) (*Spider, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	dict := make(map[string]*SksNode)
	decoder := json.NewDecoder(fh)
	err = decoder.Decode(&dict)
	if err != nil {
		return nil, err
	}

	fakeSpider := new(Spider)
	fakeSpider.serverInfos = make(map[string]*SksNode, len(dict))
	for n := range dict {
		dict[n].initialised = true
		fakeSpider.serverInfos[n] = dict[n]
	}
	return fakeSpider, nil
}
