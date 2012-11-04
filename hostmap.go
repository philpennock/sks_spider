package sks_spider

import (
	"sort"
	"strings"
)

type HostMap map[string]*SksNode

type sortingHost struct {
	reversed string
	normal   string
}
type sortingHosts []*sortingHost

func (p sortingHosts) Len() int      { return len(p) }
func (p sortingHosts) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p sortingHosts) Less(i, j int) bool {
	return p[i].reversed < p[j].reversed
}

func ReverseStringSlice(a []string) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}

// Sort a list of strings in host order, ie by DNS label from right to left
func HostSort(victim []string) {
	keyed := make(sortingHosts, len(victim))
	for i := range victim {
		keyed[i] = new(sortingHost)
		keyed[i].normal = victim[i]
		t := strings.Split(victim[i], ".")
		ReverseStringSlice(t)
		keyed[i].reversed = strings.Join(t, ".")
	}
	sort.Sort(keyed)
	for i := range victim {
		victim[i] = keyed[i].normal
	}
}

func GenerateHostlistSorted(hostMap HostMap) []string {
	hostnames := make([]string, len(hostMap))
	var i = 0
	for h := range hostMap {
		hostnames[i] = h
		i += 1
	}
	hostnames = hostnames[:i]
	HostSort(hostnames)
	return hostnames
}

func GeneratePersistedInformation(spider *Spider) *PersistedHostInfo {
	hostMap := make(HostMap, len(spider.serverInfos))
	for hn := range spider.serverInfos {
		if spider.serverInfos[hn] == nil {
			continue
		}
		hostMap[hn] = spider.serverInfos[hn]
	}

	hostnames := GenerateHostlistSorted(hostMap)

	for _, hostname := range hostnames {
		hostMap[hostname].IpList = spider.ipsForHost[hostname]
		hostMap[hostname].Aliases = make([]string, 0, len(spider.aliasesForHost[hostname]))
		for _, alias := range spider.aliasesForHost[hostname] {
			if alias != hostname {
				hostMap[hostname].Aliases = append(hostMap[hostname].Aliases, alias)
			}
		}
		HostSort(hostMap[hostname].GossipPeerList)
		HostSort(hostMap[hostname].MailsyncPeers)
	}

	// TODO: spawn go-routines, wait, to do Geo resolution
	return &PersistedHostInfo{
		HostMap: hostMap,
		Sorted:  hostnames,
		Graph:   GenerateGraph(hostnames, hostMap),
	}
}
