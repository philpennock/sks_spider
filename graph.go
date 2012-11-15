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
	btree "github.com/runningwild/go-btree"
	// gotgo
	// in-dir: gotgo -o btree.go btree.got string
	// top: go install github.com/runningwild/go-btree
)

// This is not memory efficient but for this few hosts, does not need to be

type HostGraph struct {
	maxLen   int
	outbound map[string]btree.SortedSet
	inbound  map[string]btree.SortedSet
}

func btreeStringLess(a, b string) bool {
	return a < b
}

func NewHostGraph(count int) *HostGraph {
	outbound := make(map[string]btree.SortedSet, count)
	inbound := make(map[string]btree.SortedSet, count)
	return &HostGraph{maxLen: count, outbound: outbound, inbound: inbound}
}

func (hg *HostGraph) addHost(name string, info *SksNode) {
	if _, ok := hg.outbound[name]; !ok {
		hg.outbound[name] = btree.NewTree(btreeStringLess)
	}
	if _, ok := hg.inbound[name]; !ok {
		hg.inbound[name] = btree.NewTree(btreeStringLess)
	}
	for _, host := range info.GossipPeerList {
		hg.outbound[name].Insert(host)
		if _, ok := hg.inbound[host]; !ok {
			hg.inbound[host] = btree.NewTree(btreeStringLess)
		}
		hg.inbound[host].Insert(name)
	}
}

// inbounds can exist where there's no outbound because servers are down and we just have links to them
// I don't want to deal with nil's elsewhere
func (hg *HostGraph) fixOutbounds() {
	for k := range hg.inbound {
		for hn := range hg.inbound[k].Data() {
			if _, ok := hg.outbound[hn]; !ok {
				hg.outbound[hn] = btree.NewTree(btreeStringLess)
			}
		}
	}
}

func (hg *HostGraph) Outbound(name string) <-chan string {
	return hg.outbound[name].Data()
}

func (hg *HostGraph) Inbound(name string) <-chan string {
	return hg.inbound[name].Data()
}

func (hg *HostGraph) ExistsLink(from, to string) bool {
	return hg.inbound[to].Contains(from)
}

func (hg *HostGraph) Len() int {
	l1 := len(hg.outbound)
	l2 := len(hg.inbound)
	if l1 >= l2 {
		return l1
	}
	return l2
}

func GenerateGraph(names []string, sksnodes HostMap) *HostGraph {
	graph := NewHostGraph(len(names))
	for _, hn := range names {
		graph.addHost(hn, sksnodes[hn])
	}
	graph.fixOutbounds()
	return graph
}
