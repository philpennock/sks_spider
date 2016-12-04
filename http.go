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
	"encoding/json"
	"expvar"
	"fmt"
	"html/template"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"
)

const SERVE_PREFIX = "/sks-peers"

const (
	kHTML_FAVICON = "/favicon.ico"
	kBUCKET_SIZE  = 3000
)

const (
	ContentTypeTextPlain = "text/plain; charset=UTF-8"
	ContentTypeJson      = "application/json"
)

var (
	statsCollectionTimestamp  *expvar.Int
	statsServersTotal         *expvar.Int
	statsServersHostnamesSeen *expvar.Int
	statsServersHaveData      *expvar.Int
	statsServersBadDNS        *expvar.Int
	statsServersBadData       *expvar.Int
)

func init() {
	statsCollectionTimestamp = expvar.NewInt("collection.timestamp.activated")
	statsServersTotal = expvar.NewInt("collection.servers.total")
	statsServersHostnamesSeen = expvar.NewInt("collection.servers.hostnamesseen")
	statsServersHaveData = expvar.NewInt("collection.servers.havedata")
	statsServersBadDNS = expvar.NewInt("collection.servers.baddns")
	statsServersBadData = expvar.NewInt("collection.servers.baddata")
}

func setupHttpServer(listen string) *http.Server {
	s := &http.Server{
		Addr:           listen,
		Handler:        http.DefaultServeMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 14, // we don't POST, so 16kB should be plenty (famous last words)
	}

	http.HandleFunc(SERVE_PREFIX, apiPeersPage)
	http.HandleFunc(SERVE_PREFIX+"/peer-info", apiPeerInfoPage)
	http.HandleFunc(SERVE_PREFIX+"/ip-valid", apiIpValidPage)
	http.HandleFunc(SERVE_PREFIX+"/ip-valid-stats", apiIpValidStatsPage)
	http.HandleFunc(SERVE_PREFIX+"/hostnames-json", apiHostnamesJsonPage)
	http.HandleFunc(SERVE_PREFIX+"/graph-dot", apiGraphDot)
	http.HandleFunc("/helpz", apiHelpz)
	http.HandleFunc("/scanstatusz", apiScanStatusz)
	// net/http/pprof provides /debug/pprof with threads and profiling information
	// expvar provides /debug/vars (JSON)
	// MISSING: environz rescanz (internalz) quitz
	// leave quitz out?
	http.HandleFunc("/", apiOops)
	return s
}

func apiOops(w http.ResponseWriter, req *http.Request) {
	if len(req.RequestURI) > 1 {
		http.NotFound(w, req)
		return
	}
	w.Header().Set("Content-Type", ContentTypeTextPlain)
	fmt.Fprintf(w, "You shouldn't see this top level.  Err, oops?\n")
}

func apiHelpz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", ContentTypeTextPlain)
	fmt.Fprintf(w, "Some help here one day, maybe.\n")
	//TODO: write
}

func apiScanStatusz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", ContentTypeTextPlain)
	SpiderDiagnostics(w)
	fmt.Fprintf(w, "\nDone.\n")
}

func apiPeersPage(w http.ResponseWriter, req *http.Request) {
	//TODO: restore "in progress" reporting
	//TODO: restore this as trigger for rescan if membership file has changed?
	//TODO: restore distance
	//TODO: restore entries which are missing DNS but are configured
	persisted := GetCurrentPersisted()
	var warning string
	var display_order = []string{}
	if persisted == nil {
		warning = "Still awaiting data collection"
	} else {
		display_order = persisted.DepthSorted
	}

	namespace := genNamespace()
	namespace["Scanning_active"] = ""

	// IsZero will hold if persisted loaded from JSON which predates change
	// that adds the timestamp.
	if persisted != nil && !persisted.Timestamp.IsZero() {
		namespace["LastScanTime"] = persisted.Timestamp.UTC().Format("20060102_150405") + "Z"
	}

	namespace["Mesh_count"] = len(display_order)
	if len(display_order) > 0 {
		pc := 0
		for _, name := range display_order {
			d := persisted.HostMap[name].Distance
			if d == 1 {
				pc += 1
			} else if d > 1 {
				break
			}
		}
		namespace["Peer_count"] = pc
	} else {
		namespace["Peer_count"] = 0
	}

	if warning != "" {
		namespace["warning"] = warning
	}
	serveTemplates["head"].Execute(w, namespace)

	for index, host := range display_order {
		node := persisted.HostMap[host]
		attributes := make(map[string]interface{}, 11)
		if len(node.IpList) > 1 {
			attributes["Rowspan"] = template.HTMLAttr(fmt.Sprintf(" rowspan=\"%d\"", len(node.IpList)))
		} else {
			attributes["Rowspan"] = template.HTMLAttr("")
		}
		if index%2 == 0 {
			attributes["Rowclass"] = "even"
		} else {
			attributes["Rowclass"] = "odd"
		}
		attributes["Hostname"] = host
		attributes["Sks_info"] = NodeUrl(host, node)
		attributes["Info_page"] = fmt.Sprintf(SERVE_PREFIX+"/peer-info?peer=%s", host)

		if node.AnalyzeError != "" {
			attributes["Error"] = node.AnalyzeError
			serveTemplates["hosterr"].Execute(w, attributes)
			continue
		}

		switch node.Distance {
		case 0:
			attributes["Mutual"] = "n/a"
		case 1:
			attributes["Mutual"] = persisted.Graph.LabelMutualWithBase(host)
		default:
			attributes["Mutual"] = "-"
		}
		if len(node.Aliases) > 0 {
			attributes["Host_aliases_text"] = template.HTML(fmt.Sprintf(" <span class=\"host_aliases\">%s</span>", node.Aliases))
		} else {
			attributes["Host_aliases_text"] = ""
		}
		attributes["Version"] = node.Version
		attributes["Keycount"] = node.Keycount
		attributes["Distance"] = node.Distance
		attributes["Web_server"] = node.ServerHeader
		if node.ViaHeader != "" {
			attributes["Via_info"] = fmt.Sprintf("✓ [%s]", node.ViaHeader)
		} else {
			attributes["Via_info"] = "✗"
		}
		for n, ip := range node.IpList {
			attributes["Ip"] = ip
			attributes["Geo"] = persisted.IPCountryMap[ip]
			if n == 0 {
				serveTemplates["host"].Execute(w, attributes)
			} else {
				serveTemplates["hostmore"].Execute(w, attributes)
			}
		}
	}

	serveTemplates["foot"].Execute(w, namespace)
}

func apiPeerInfoPage(w http.ResponseWriter, req *http.Request) {
	var err error
	if err = req.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form information", http.StatusBadRequest)
		return
	}
	peer := req.Form.Get("peer")
	if peer == "" {
		http.Error(w, "Missing 'peer' parameter to query", http.StatusBadRequest)
		return
	}

	namespace := genNamespace()
	namespace["Peername"] = peer
	persisted := GetCurrentPersisted()
	var warning string
	var node *SksNode
	var ok bool

	if persisted == nil {
		warning = "Still awaiting data collection"
	} else if node, ok = persisted.HostMap[peer]; !ok {
		warning = fmt.Sprintf("Peer \"%s\" not found", peer)
	}

	if warning != "" {
		namespace["Warning"] = warning
		serveTemplates["pi_head"].Execute(w, namespace)
		serveTemplates["pi_foot"].Execute(w, namespace)
		return
	}

	namespace["Keycount"] = node.Keycount
	namespace["Version"] = node.Version
	if node.Software != "" {
		namespace["Software"] = node.Software
	} else {
		namespace["Software"] = defaultSoftware
	}
	namespace["Ips"] = "[" + strings.Join(node.IpList, "], [") + "]"
	namespace["Mailsync"] = node.MailsyncPeers
	namespace["Mailsync_count"] = len(node.MailsyncPeers)
	namespace["Web_server"] = node.ServerHeader
	namespace["Via_info"] = node.ViaHeader
	namespace["Peer_statsurl"] = node.Url()

	peer_list := persisted.Graph.AllPeersOf(node.Hostname)

	serveTemplates["pi_head"].Execute(w, namespace)
	serveTemplates["pi_main"].Execute(w, namespace)
	serveTemplates["pi_peers_start"].Execute(w, namespace)

	for _, other := range peer_list {
		attributes := make(map[string]interface{}, 5)
		attributes["Name"] = other
		attributes["Ref_url"] = NodeUrl(other, persisted.HostMap[other])
		out := persisted.Graph.ExistsLink(peer, other)
		in := persisted.Graph.ExistsLink(other, peer)
		common := out && in
		attributes["Out"] = out
		if _, ok := persisted.AliasMap[other]; !ok {
			// peer not successfully polled
			attributes["In"] = "?"
			attributes["Common"] = "?"
		} else {
			attributes["In"] = in
			attributes["Common"] = common
		}
		serveTemplates["pi_peers"].Execute(w, attributes)
	}

	serveTemplates["pi_peers_end"].Execute(w, namespace)
	serveTemplates["pi_foot"].Execute(w, namespace)
}

func apiHostnamesJsonPage(w http.ResponseWriter, req *http.Request) {
	var err error
	if err = req.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form information", http.StatusBadRequest)
		return
	}
	all := false
	if _, ok := req.Form["all"]; ok {
		all = true
	} else if _, ok := req.Form["mesh"]; ok {
		all = true
	}

	var hostList []string

	if all {
		hosts := GetCurrentHosts()
		if hosts == nil || len(hosts) == 0 {
			Log.Printf("Request for current hosts, none loaded yet")
			http.Error(w, "Still waiting for data collection", http.StatusServiceUnavailable)
			return
		}
		hostList = make([]string, len(hosts))
		i := 0
		for k := range hosts {
			hostList[i] = k
			i += 1
		}
	} else {
		hostList, err = GetMembershipHosts()
		if err != nil {
			Log.Printf("Failed to load membership: %s", err)
			http.Error(w, "Problem loading membership file", http.StatusServiceUnavailable)
			return
		}
	}

	b, err := json.Marshal(hostList)
	if err != nil {
		Log.Printf("Failed to marshal hostlist to JSON: %s", err)
		http.Error(w, "JSON encoding glitch", http.StatusInternalServerError)
		return
	}

	contentType := ContentTypeJson
	if _, ok := req.Form["textplain"]; ok {
		contentType = ContentTypeTextPlain
	}
	w.Header().Set("Content-Type", contentType)
	fmt.Fprintf(w, "{ \"hostnames\": %s }\n", b)
}
