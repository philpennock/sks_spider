/*
   Copyright 2009-2012,2016 Phil Pennock

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
	"strings"
)

// This is not memory efficient but for this few hosts, does not need to be

type HostGraph struct {
	maxLen   int
	aliases  AliasMap
	outbound collectionOfSortedSets
	inbound  collectionOfSortedSets
}

func NewHostGraph(count int, aliasMap AliasMap) *HostGraph {
	return &HostGraph{
		maxLen:   count,
		aliases:  aliasMap,
		outbound: newCollectionOfSortedSets(count),
		inbound:  newCollectionOfSortedSets(count),
	}
}

func (hg *HostGraph) addHost(name string, info *SksNode) {
	hg.outbound.Ensure(name)
	hg.inbound.Ensure(name)
	for _, peerAsGiven := range info.GossipPeerList {
		var peerCanonical string
		if canon, ok := hg.aliases[strings.ToLower(peerAsGiven)]; ok {
			peerCanonical = canon
		} else {
			lowered := strings.ToLower(peerAsGiven)
			peerCanonical = lowered
			// peer is down, have no node, but still have outbound link:
			hg.aliases[lowered] = lowered
			if peerAsGiven != lowered {
				hg.aliases[peerAsGiven] = lowered
			}
		}
		hg.outbound.Insert(name, peerCanonical)
		hg.inbound.Insert(peerCanonical, name)
	}
}

// inbounds can exist where there's no outbound because servers are down and we just have links to them
// I don't want to deal with nil's elsewhere
func (hg *HostGraph) fixOutbounds() {
	for k := range hg.inbound {
		for _, hn := range hg.inbound[k].AllData() {
			hg.outbound.Ensure(hn)
		}
	}
}

func (hg *HostGraph) Outbound(name string) <-chan string {
	return hg.outbound[strings.ToLower(name)].Data()
}

func (hg *HostGraph) Inbound(name string) <-chan string {
	return hg.inbound[strings.ToLower(name)].Data()
}

func (hg *HostGraph) ExistsLink(from, to string) bool {
	realFrom, okFrom := hg.aliases[strings.ToLower(from)]
	realTo, okTo := hg.aliases[strings.ToLower(to)]
	if !okFrom || !okTo {
		Log.Printf("Bad link query, internal bug: %s %v -> %s %v", from, okFrom, to, okTo)
		return false
	}
	return hg.inbound[realTo].Contains(realFrom)
}

func (hg *HostGraph) AllPeersOf(name string) []string {
	canonName, ok := hg.aliases[strings.ToLower(name)]
	if !ok {
		return []string{}
	}
	allPeers := newHostReversedSet()
	if _, ok := hg.outbound[canonName]; ok {
		for out := range hg.outbound[canonName].Data() {
			allPeers.Insert(out)
		}
	} else {
		Log.Printf("Warning: missing hostgraph outbound for %q", canonName)
	}
	if _, ok := hg.inbound[canonName]; ok {
		for in := range hg.inbound[canonName].Data() {
			allPeers.Insert(in)
		}
	} else {
		Log.Printf("Warning: missing hostgraph inbound for %q", canonName)
	}
	sortedList := make([]string, allPeers.Len())
	i := 0
	for peer := range allPeers.Data() {
		sortedList[i] = peer
		i++
	}
	return sortedList
}

func (hg *HostGraph) Len() int {
	l1 := len(hg.outbound)
	l2 := len(hg.inbound)
	if l1 >= l2 {
		return l1
	}
	return l2
}

func GenerateGraph(names []string, sksnodes HostMap, aliases AliasMap) *HostGraph {
	graph := NewHostGraph(len(names), aliases)
	for _, hn := range names {
		hnLower := strings.ToLower(hn)
		graph.addHost(hnLower, sksnodes[hn])
	}
	graph.fixOutbounds()
	return graph
}

func (hg *HostGraph) LabelMutualWithBase(name string) string {
	baseCanon, ok := hg.aliases[*flSpiderStartHost]
	if !ok {
		panic("no known alias for start host")
	}
	canon, ok := hg.aliases[name]
	switch {
	case !ok:
		// can't be mutual, we don't even know the name
		return "No"
	case canon == baseCanon:
		return "n/a"
	case hg.ExistsLink(canon, baseCanon) && hg.ExistsLink(baseCanon, canon):
		return "Yes"
	default:
		return "No"
	}
}
