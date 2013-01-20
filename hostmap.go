/*
   Copyright 2009-2013 Phil Pennock

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
	"sort"
	"strings"
)

type HostMap map[string]*SksNode
type AliasMap map[string]string
type IPCountryMap map[string]string

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

func GetAliasMapForHostmap(hostMap HostMap) AliasMap {
	aliasMap := make(AliasMap, len(hostMap)*2)
	for hostname, node := range hostMap {
		aliasMap[hostname] = hostname
		for _, alias := range node.Aliases {
			aliasMap[alias] = hostname
		}
	}
	return aliasMap
}

func GeneratePersistedInformation(spider *Spider) *PersistedHostInfo {
	hostMap := make(HostMap, len(spider.serverInfos))
	aliasMap := make(AliasMap, len(spider.serverInfos)*2)
	for hn := range spider.serverInfos {
		if spider.serverInfos[hn] == nil {
			continue
		}
		hostMap[hn] = spider.serverInfos[hn]
	}

	hostnames := GenerateHostlistSorted(hostMap)

	for _, hostname := range hostnames {
		aliasMap[hostname] = hostname
		hostMap[hostname].IpList = spider.ipsForHost[hostname]
		hostMap[hostname].Aliases = make([]string, 0, len(spider.aliasesForHost[hostname]))
		for _, alias := range spider.aliasesForHost[hostname] {
			aliasMap[alias] = hostname
			if alias != hostname {
				hostMap[hostname].Aliases = append(hostMap[hostname].Aliases, alias)
			}
		}
		HostSort(hostMap[hostname].GossipPeerList)
		HostSort(hostMap[hostname].MailsyncPeers)
		hostMap[hostname].Distance = spider.distances[hostname]
		// To let JSON Marshal/Unmarshal work:
		if hostMap[hostname].analyzeError != nil {
			hostMap[hostname].AnalyzeError = hostMap[hostname].analyzeError.Error()
			hostMap[hostname].analyzeError = nil
		}
	}

	countryMap := make(IPCountryMap, len(spider.countriesForIPs))
	for ip, country := range spider.countriesForIPs {
		if country != "" {
			countryMap[ip] = country
		}
	}

	// TODO: spawn go-routines, wait, to do Geo resolution
	return &PersistedHostInfo{
		HostMap:      hostMap,
		AliasMap:     aliasMap,
		IPCountryMap: countryMap,
		Sorted:       hostnames,
		DepthSorted:  GenerateDepthSorted(hostMap),
		Graph:        GenerateGraph(hostnames, hostMap, aliasMap),
	}
}

func GetFreshCountryForHostmap(hostMap HostMap) IPCountryMap {
	Log.Print("Quering DNS (sequentially) for fresh country map")
	countryMap := make(IPCountryMap, len(hostMap))
	triedIPs := make(map[string]bool, len(hostMap)*3)
	for _, node := range hostMap {
		if node.IpList == nil {
			continue
		}
		for _, ip := range node.IpList {
			if _, seen := triedIPs[ip]; seen {
				continue
			}
			triedIPs[ip] = true
			country, err := CountryForIPString(ip)
			if err == nil {
				countryMap[ip] = country
			}
		}
	}
	Log.Printf("Got countries for %d (of %d) IPs", len(countryMap), len(triedIPs))
	return countryMap
}

func (p *PersistedHostInfo) LogInformation() {
	Log.Printf("Persisting: sizes HostMap=%d AliasMap=%d IPCountryMap=%d Sorted=%d DepthSorted=%d Graph=%d",
		len(p.HostMap), len(p.AliasMap), len(p.IPCountryMap),
		len(p.Sorted), len(p.DepthSorted), p.Graph.Len())
}

func (p *PersistedHostInfo) UpdateStatsCounters(spider *Spider) {
	statsCollectionTimestamp.Set(p.Timestamp.Unix())
	var countOkayAndBad = int64(len(p.HostMap))
	var countBadData int64 = 0
	for hostname := range p.HostMap {
		if p.HostMap[hostname].AnalyzeError != "" {
			countBadData++
		}
	}
	statsServersHaveData.Set(countOkayAndBad - countBadData)
	statsServersBadData.Set(countBadData)
	statsServersBadDNS.Set(int64(len(spider.badDNS)))
	statsServersTotal.Set(int64(len(p.HostMap)))
	statsServersHostnamesSeen.Set(int64(len(spider.considering)))
}
