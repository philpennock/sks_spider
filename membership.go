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
