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
	"html/template"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

import (
	btree "github.com/runningwild/go-btree"
)

const SERVE_PREFIX = "/sks-peers"

const (
	kHTML_CSS_STYLESHEET = "/style/spodhuis-sks.css"
	kHTML_FAVICON        = "/favicon.ico"
	kBUCKET_SIZE         = 3000
)

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
	// MISSING: graph-dot
	http.HandleFunc("/helpz", apiHelpz)
	http.HandleFunc("/scanstatusz", apiScanStatusz)
	// MISSING: threadz environz rescanz internalz
	http.HandleFunc("/", apiOops)
	return s
}

func apiOops(w http.ResponseWriter, req *http.Request) {
	if len(req.RequestURI) > 1 {
		http.NotFound(w, req)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	fmt.Fprintf(w, "You shouldn't see this top level.  Err, oops?\n")
}

func apiHelpz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	fmt.Fprintf(w, "Some help here one day, maybe.\n")
	//TODO: write
}

func apiScanStatusz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
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
			serveTemplates["hosterr"].Execute(w, namespace)
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

func apiIpValidPage(w http.ResponseWriter, req *http.Request) {
	var err error
	if err = req.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form information", http.StatusBadRequest)
		return
	}
	var (
		showStats        bool
		emitJson         bool
		limitToProxies   bool
		limitToCountries *CountrySet
	)
	if _, ok := req.Form["stats"]; ok {
		showStats = true
	}
	if _, ok := req.Form["json"]; ok {
		emitJson = true
	}
	if _, ok := req.Form["proxies"]; ok {
		limitToProxies = true
	}
	if _, ok := req.Form["countries"]; ok {
		limitToCountries = NewCountrySet(req.Form.Get("countries"))
	}

	statsList := make([]string, 0, 100)
	Statsf := func(s string, v ...interface{}) {
		statsList = append(statsList, fmt.Sprintf(s, v...))
	}

	var (
		abortMessage func(string)
		doShowStats  func()
		contentType  string
	)

	if emitJson {
		contentType = "application/json"
		doShowStats = func() {
			b, err := json.Marshal(statsList)
			if err != nil {
				Log.Printf("Unable to JSON marshal stats: %s", err)
				return
			}
			fmt.Fprintf(w, "\"stats\": %s\n", b)
		}
		abortMessage = func(s string) {
			fmt.Fprintf(w, "{\n")
			if showStats {
				doShowStats()
				fmt.Fprintf(w, ", ")
			}
			fmt.Fprintf(w, `"status": { "status": "INVALID", "count": 0, "reason": "%s" }`, s)
			fmt.Fprintf(w, "\n}\n")
		}
	} else {
		contentType = "text/plain; charset=UTF-8"
		doShowStats = func() {
			for _, l := range statsList {
				fmt.Fprintf(w, "STATS: %s\n", l)
			}
		}
		abortMessage = func(s string) {
			if showStats {
				doShowStats()
			}
			fmt.Fprintf(w, "IP-Gen/1.1: status=INVALID count=0 reason=%s\n.\n", s)
		}
	}
	w.Header().Set("Content-Type", contentType)

	persisted := GetCurrentPersisted()
	if persisted == nil {
		abortMessage("first_scan")
		return
	}

	var minimumVersion *SksVersion = nil
	mvReq := req.Form.Get("minimum_version")
	if mvReq != "" {
		tmp := NewSksVersion(mvReq)
		minimumVersion = tmp
	}

	var (
		// for stats, we avoid double-weighting dual-stack boxes by working with
		// just one IP per box, but then later deal with all the IPs for filtering.
		ips_one_per_server = make(map[string]int, len(persisted.HostMap)*2)
		ips_all            = make(map[string]int, len(persisted.HostMap)*2)
	)

	var (
		count_servers_1010            int
		count_servers_too_old         int
		count_servers_unwanted_server int
		count_servers_wrong_country   int
		ips_skip_1010                 btree.SortedSet = btree.NewTree(btreeStringLess)
		ips_too_old                   btree.SortedSet = btree.NewTree(btreeStringLess)
		ips_unwanted_server           btree.SortedSet = btree.NewTree(btreeStringLess)
		ips_wrong_country             btree.SortedSet = btree.NewTree(btreeStringLess)
	)

	for _, name := range persisted.Sorted {
		node := persisted.HostMap[name]
		var (
			skip_this_1010     = false
			skip_this_age      = false
			skip_this_nonproxy = false
			skip_this_country  = false
		)
		if node.Keycount <= 1 {
			Statsf("dropping server <%s> with %d keys", name, node.Keycount)
			continue
		}

		if string(node.Version) == "1.0.10" {
			skip_this_1010 = true
			//ips_skip_1010.Insert(name) // nope, IPs
			count_servers_1010 += 1
		}

		if minimumVersion != nil {
			thisVersion := NewSksVersion(node.Version)
			if thisVersion == nil || !thisVersion.IsAtLeast(minimumVersion) {
				skip_this_age = true
				count_servers_too_old += 1
			}
		}

		if limitToProxies && node.ViaHeader == "" {
			server := strings.ToLower(strings.SplitN(node.ServerHeader, "/", 2)[0])
			if _, ok := serverHeadersNative[server]; ok {
				skip_this_nonproxy = true
				count_servers_unwanted_server += 1
			}
		}

		if limitToCountries != nil {
			//XXX collect node countries here, yada yada
			// for now, if it's a country constraint, nothing matches
			skip_this_country = true
			count_servers_wrong_country += 1
		}

		if len(node.IpList) > 0 {
			ips_one_per_server[node.IpList[0]] = node.Keycount
			for _, ip := range node.IpList {
				ips_all[ip] = node.Keycount
				if skip_this_1010 {
					ips_skip_1010.Insert(ip)
				}
				if skip_this_age {
					ips_too_old.Insert(ip)
				}
				if skip_this_nonproxy {
					ips_unwanted_server.Insert(ip)
				}
				if skip_this_country {
					ips_wrong_country.Insert(ip)
				}
			}
		}

	}

	// We want to discard statistic-distorting outliers, then of what remains,
	// discard those too far away from "normal", but we really want the "best"
	// servers to be our guide, so 1 std-dev of the second-highest remaining
	// value should be safe; in fact, we'll hardcode a limit of how far below.
	// To discard, find mode size (knowing that value can be split across two
	// buckets) and discard more than five stddevs from mode.  The bucketing
	// should be larger than the distance from desired value so that the mode
	// is only split across two buckets, if we assume enough servers that a
	// small number will be down, most will be valid-if-large-enough, so that
	// splitting the count across two buckets won't let the third-best value win

	// This is barely-modified from Python, just enough to translate language, not idioms
	// This was ... "much easier" with list comprehensions in Python
	var buckets = make(map[int][]int, 40)
	for _, count := range ips_one_per_server {
		bucket := int(count / kBUCKET_SIZE)
		if _, ok := buckets[bucket]; !ok {
			buckets[bucket] = make([]int, 0, 20)
		}
		buckets[bucket] = append(buckets[bucket], count)
	}
	if len(buckets) == 0 {
		abortMessage("broken_no_buckets")
		return
	}

	var largest_bucket int
	var largest_bucket_len int
	for k := range buckets {
		if len(buckets[k]) > largest_bucket_len {
			largest_bucket = k
			largest_bucket_len = len(buckets[k])
		}
	}
	first_n := len(buckets[largest_bucket])
	var first_sum int
	for _, v := range buckets[largest_bucket] {
		first_sum += v
	}
	first_mean := float64(first_sum) / float64(first_n)
	var first_sd float64
	for _, v := range buckets[largest_bucket] {
		d := float64(v) - first_mean
		first_sd += d * d
	}
	first_sd = math.Sqrt(first_sd / float64(first_n))
	first_bounds_min := int(first_mean - 5*first_sd)
	first_bounds_max := int(first_mean + 5*first_sd)

	first_ips_list := make([]string, 0, len(ips_one_per_server))
	for ip := range ips_one_per_server {
		if first_bounds_min <= ips_all[ip] && ips_all[ip] <= first_bounds_max {
			first_ips_list = append(first_ips_list, ip)
		}
	}
	first_ips_alllist := make([]string, 0, len(ips_all))
	for ip := range ips_all {
		if first_bounds_min <= ips_all[ip] && ips_all[ip] <= first_bounds_max {
			first_ips_alllist = append(first_ips_alllist, ip)
		}
	}
	var second_mean, second_sd float64
	first_ips := make(map[string]int, len(first_ips_list))
	for _, ip := range first_ips_list {
		first_ips[ip] = ips_all[ip]
		second_mean += float64(ips_all[ip])
	}
	first_ips_all := make(map[string]int, len(first_ips_alllist))
	for _, ip := range first_ips_alllist {
		first_ips_all[ip] = ips_all[ip]
	}
	second_mean /= float64(len(first_ips_list))
	for _, v := range first_ips {
		d := float64(v) - second_mean
		second_sd += d * d
	}
	second_sd = math.Sqrt(second_sd / float64(len(first_ips_list)))

	if showStats {
		Statsf("have %d servers in %d buckets (%d ips total)", len(ips_one_per_server), len(buckets), len(ips_all))
		bucket_sizes := make([]int, 0, len(buckets))
		for k := range buckets {
			bucket_sizes = append(bucket_sizes, k)
		}
		sort.Ints(bucket_sizes)
		for _, b := range bucket_sizes {
			Statsf("%6d: %s", b, strings.Repeat("*", len(buckets[b])))
		}
		Statsf("largest bucket is %d with %d entries", largest_bucket, first_n)
		Statsf("bucket size %d means bucket %d is [%d, %d)", kBUCKET_SIZE, largest_bucket,
			kBUCKET_SIZE*largest_bucket, kBUCKET_SIZE*(largest_bucket+1))
		Statsf("largest bucket: mean=%f sd=%f", first_mean, first_sd)
		Statsf("first bounds: [%d, %d]", first_bounds_min, first_bounds_max)
		Statsf("have %d servers within bounds, mean value %f sd=%f", len(first_ips_list), second_mean, second_sd)
	}

	if second_mean < float64(*flKeysSanityMin) {
		Statsf("mean %f < %d", second_mean, *flKeysSanityMin)
		abortMessage("broken_data")
		return
	}
	threshold_base_index := len(first_ips) - 2
	if threshold_base_index < 0 {
		threshold_base_index = 0
	}
	threshold_candidates := make([]int, 0, len(first_ips))
	for _, count := range first_ips {
		threshold_candidates = append(threshold_candidates, count)
	}
	sort.Ints(threshold_candidates)
	var threshold int = threshold_candidates[threshold_base_index] - (*flKeysDailyJitter + int(second_sd))

	if showStats {
		Statsf("Second largest count within bounds: %d", threshold_candidates[threshold_base_index])
		Statsf("threshold: %d", threshold)
	}

	if nt, ok := req.Form["threshold"]; ok {
		i, ok2 := strconv.Atoi(nt[0])
		if ok2 == nil && i > 0 {
			Statsf("Overriding threshold from CGI parameter; %d -> %d", threshold, i)
			threshold = i
		}
	}

	ips := make([]string, 0, len(first_ips_all))
	for ip, count := range first_ips_all {
		if count >= threshold {
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		Statsf("No IPs above threshold %d", threshold)
		abortMessage("threshold_too_high")
		return
	}

	filterOut := func(rationale string, eliminate btree.SortedSet, eliminate_server_count int, candidates []string) []string {
		alreadyDropped := btree.NewTree(btreeStringLess)
		for ip := range eliminate.Data() {
			alreadyDropped.Insert(ip)
		}
		for _, ip := range candidates {
			alreadyDropped.Remove(ip)
		}
		ips = make([]string, 0, len(candidates))
		for _, ip := range candidates {
			if !eliminate.Contains(ip) {
				ips = append(ips, ip)
			}
		}
		Statsf("dropping all %d servers %s, for %d possible IPs but %d of those already dropped",
			eliminate_server_count, rationale, eliminate.Len(), alreadyDropped.Len())
		return ips
	}

	ips = filterOut("running version v1.0.10", ips_skip_1010, count_servers_1010, ips)
	if len(ips) == 0 {
		abortMessage("No_servers_left_after_v1.0.10_filter")
		return
	}

	if minimumVersion != nil {
		ips = filterOut(fmt.Sprintf("running version < v%s", minimumVersion), ips_too_old, count_servers_too_old, ips)
		if len(ips) == 0 {
			abortMessage(fmt.Sprintf("No_servers_left_after_minimum_version_filter_(v%s)", minimumVersion))
			return
		}
	}

	if limitToCountries != nil {
		ips = filterOut(fmt.Sprintf("not in countries [%s]", limitToCountries), ips_wrong_country, count_servers_wrong_country, ips)
		if len(ips) == 0 {
			abortMessage(fmt.Sprintf("No_servers_left_after_country_filter_[%s]", limitToCountries))
			return
		}
	}

	if limitToProxies {
		ips = filterOut("not behind a web-proxy", ips_unwanted_server, count_servers_unwanted_server, ips)
		if len(ips) == 0 {
			abortMessage("No_servers_left_after_proxies_filter")
			return
		}
	}

	//TODO: change now to be the time the scan finished
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05") + "Z"
	count := len(ips)
	Log.Printf("ip-valid: Yielding %d of %d values", count, len(ips_all))

	// The tags are public statements; history:
	//   skip 1.0.10 -> skip_1010, because of lookup problems biting gnupg
	//   alg_1 used a fixed threshold (too small to deal with jitter)
	//   alg_2 used stddev+jitter
	//   alg_3 fixed maximum bucket selection (was a code bug)
	//   alg_4 stopped double-counting servers with multiple IP addresses
	//   alg_5 keep 1.0.10 servers for long enough to calculate stats, drop afterwards
	statusD := make(map[string]interface{}, 16)
	statusD["status"] = "COMPLETE"
	statusD["count"] = count
	statusD["tags"] = []string{"skip_1010", "alg_5"}
	if minimumVersion != nil {
		statusD["minimum_version"] = minimumVersion.String()
	}
	if limitToProxies {
		statusD["proxies"] = "1"
	}
	if limitToCountries != nil {
		statusD["countries"] = limitToCountries.String()
	}
	statusD["minimum"] = threshold
	statusD["collected"] = timestamp

	if emitJson {
		fmt.Fprintf(w, "{\n")
		if showStats {
			doShowStats()
			fmt.Fprintf(w, ", ")
		}
		bIps, _ := json.Marshal(ips)
		bStatus, _ := json.Marshal(statusD)
		fmt.Fprintf(w, "\"status\": %s,\n\"ips\": %s\n}\n", bStatus, bIps)
	} else {
		if showStats {
			doShowStats()
		}
		fmt.Fprintf(w, "IP-Gen/1.1:")
		for k, v := range statusD {
			var vstr string
			//fmt.Fprintf(w, " {{%T}}", v)
			switch v.(type) {
			case int:
				vstr = strconv.Itoa(v.(int))
			case []string:
				vstr = strings.Join(v.([]string), ",")
			default:
				vstr = fmt.Sprintf("%s", v)
			}
			fmt.Fprintf(w, " %s=%s", k, vstr)
		}
		fmt.Fprintf(w, "\n")
		for _, ip := range ips {
			fmt.Fprintf(w, "%s\n", ip)
		}
		fmt.Fprintf(w, ".\n")
	}

}

func apiIpValidStatsPage(w http.ResponseWriter, req *http.Request) {
	var err error
	if err = req.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form information", http.StatusBadRequest)
		return
	}
	req.Form.Set("stats", "1")
	apiIpValidPage(w, req)
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

	contentType := "application/json"
	if _, ok := req.Form["textplain"]; ok {
		contentType = "text/plain"
	}
	w.Header().Set("Content-Type", contentType)
	fmt.Fprintf(w, "{ \"hostnames\": %s }\n", b)
}

var serveTemplates map[string]*template.Template

func genNamespace() map[string]interface{} {
	ns := make(map[string]interface{}, 50)
	ns["Maintainer"] = *flMaintEmail
	ns["MyHostname"] = *flHostname
	ns["Warning"] = ""
	return ns
}

// style here is lacking, but it's a straight C&P/translate from my older Python
func prepareTemplates() {
	kPAGE_TEMPLATE_BASIC_HEAD := fmt.Sprintf(`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">
<html>
 <head>
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
  <meta http-equiv="Content-Script-Type" content="application/ecmascript; charset=UTF-8">
  <meta http-equiv="Content-Style-Type" content="text/css; charset=UTF-8">
  <meta http-equiv="imagetoolbar" content="no"> <!-- MSIE control -->
  <link rel="stylesheet" href="%s" type="text/css" charset="utf-8">
  <link rel="shortcut icon" href="%s" type="image/x-icon">
`, kHTML_CSS_STYLESHEET, kHTML_FAVICON)

	kPAGE_TEMPLATE_BADUSER := kPAGE_TEMPLATE_BASIC_HEAD + `
  <link rev="made" href="mailto:{{.Maintainer}}">
  <title>{{.Summary}}</title>
 </head>
 <body>
  <h1>{{.Summary}}</h1>
  <div class="usererror">{{.Error}}</div>
 </body>
</html>
`

	kPAGE_TEMPLATE_HEAD := kPAGE_TEMPLATE_BASIC_HEAD + `
  <link rev="made" href="mailto:{{.Maintainer}}">
  <title>{{.MyHostname}} Peer Mesh</title>
 </head>
 <body>
  <h1>{{.MyHostname}} Peer Mesh</h1>
{{.Warning}}
{{.Scanning_active}}
  <div class="explain">
   Entries at depth 1 are direct peers.  Others are seen by spidering the peers.
   (Functionality currently limited owing to rewrite of server)
  </div>
  <table class="sks peertable">
   <thead><tr><th>Host</th><th>Info</th><th>IP</th><th>Geo</th><th>Mutual</th><th>Version</th><th>Keys</th><th>Distance</th><th>WebServer</th><th>Proxy/via</th></tr></thead>
   <tbody>
`

	kPAGE_TEMPLATE_FOOT := `
   <caption>SKS has {{.Peer_count}} peers of {{.Mesh_count}} visible</caption>
  </table>
 </body>
</html>
`

	kPAGE_TEMPLATE_HOST := `
   <tr class="peer host {{.Rowclass}}">
    <td class="hostname"{{.Rowspan}}><a href="{{.Sks_info}}">{{.Hostname}}</a>{{.Host_aliases_text}}</td>
    <td class="morelink"{{.Rowspan}}><a href="{{.Info_page}}">&dagger;</a></td>
    <td class="ipaddr">{{.Ip}}</td>
    <td class="location">{{.Geo}}</td>
    <td class="mutual"{{.Rowspan}}>{{.Mutual}}</td>
    <td class="version"{{.Rowspan}}>{{.Version}}</td>
    <td class="keys"{{.Rowspan}}>{{.Keycount}}</td>
    <td class="peer_distance"{{.Rowspan}}>{{.Distance}}</td>
    <td class="web_server"{{.Rowspan}}>{{.Web_server}}</td>
    <td class="via_proxy"{{.Rowspan}}>{{.Via_info}}</td>
   </tr>
`

	kPAGE_TEMPLATE_HOSTERR := `
   <tr class="peer host failure {{.Rowclass}}">
    <td class="hostname">{{.Hostname}}</td>
    <td class="morelink"><a href="{{.Info_page}}">&dagger;</a></td>
    <td class="exception" colspan="8">Error: {{.Error}}</td>
   </tr>
`

	kPAGE_TEMPLATE_HOSTMORE := `
   <tr class="peer more">
    <td class="ipaddr">{{.Ip}}</td><td class="location">{{.Geo}}</td>
   </tr>
`

	kPAGE_TEMPLATE_HEAD_PEER_INFO := kPAGE_TEMPLATE_BASIC_HEAD + `
  <link rev="made" href="mailto:{{.Maintainer}}">
  <title>Peer stats {{.Peername}}</title>
 </head>
 <body>
  <h1>Peer stats {{.Peername}}</h1>
{{.Warning}}
`

	kPAGE_TEMPLATE_PEER_INFO_MAIN := `
  <table class="peer_info">
   <tr><td>Name</td><td><a href="{{.Peer_statsurl}}">{{.Peername}}</a></td></tr>
   <tr><td>IPs</td><td>{{.Ips}}</td></tr>
   <tr><td>Software</td><td>{{.Software}}</td></tr>
   <tr><td>Software Version</td><td>{{.Version}}</td></tr>
   <tr><td>Web Server</td><td>{{.Web_server}}</td></tr>
   <tr><td>Proxy / via</td><td>{{.Via_info}}</td></tr>
   <tr><td>Key count</td><td>{{.Keycount}}</td></tr>
{{if .Mailsync_count}}
   <tr><td rowspan=".Mailsync_count">Mailsync</td>{{$need_tr := false}}
{{range .Mailsync}}
  {{if $need_tr}}</tr>
  <tr>{{end}}<td>{{.}}</td></tr>{{$need_tr := true}}
{{end}}
{{else}}
   <tr><td>Mailsync</td><td><em>None</em></td></tr>
{{end}}
  </table>
`

	kPAGE_TEMPLATE_PEER_INFO_PEERS_START := `
  <table class="peers">
   <caption>Peers of {{.Peername}}</caption>
   <tr><th>Name</th><th>Common</th><th>Outbound</th><th>Inbound</th></tr>
`

	// name in out common in_only out_only
	kPAGE_TEMPLATE_PEER_INFO_PEERS := `
   <tr><td><a href="{{.Ref_url}}">{{.Name}}</a></td><td>{{.Common}}</td><td>{{.Out}}</td><td>{{.In}}</td></tr>
`

	kPAGE_TEMPLATE_PEER_INFO_PEERS_END := " </table>\n"

	kPAGE_TEMPLATE_FOOT_PEER_INFO := " </body>\n</html>\n"

	serveTemplates = make(map[string]*template.Template, 16)
	serveTemplates["baduser"] = template.Must(template.New("baduser").Parse(kPAGE_TEMPLATE_BADUSER))
	serveTemplates["head"] = template.Must(template.New("head").Parse(kPAGE_TEMPLATE_HEAD))
	serveTemplates["foot"] = template.Must(template.New("foot").Parse(kPAGE_TEMPLATE_FOOT))
	serveTemplates["host"] = template.Must(template.New("host").Parse(kPAGE_TEMPLATE_HOST))
	serveTemplates["hosterr"] = template.Must(template.New("hosterr").Parse(kPAGE_TEMPLATE_HOSTERR))
	serveTemplates["hostmore"] = template.Must(template.New("hostmore").Parse(kPAGE_TEMPLATE_HOSTMORE))
	serveTemplates["pi_head"] = template.Must(template.New("pi_head").Parse(kPAGE_TEMPLATE_HEAD_PEER_INFO))
	serveTemplates["pi_main"] = template.Must(template.New("pi_main").Parse(kPAGE_TEMPLATE_PEER_INFO_MAIN))
	serveTemplates["pi_peers_start"] = template.Must(template.New("pi_peers_start").Parse(kPAGE_TEMPLATE_PEER_INFO_PEERS_START))
	serveTemplates["pi_peers"] = template.Must(template.New("pi_peers").Parse(kPAGE_TEMPLATE_PEER_INFO_PEERS))
	serveTemplates["pi_peers_end"] = template.Must(template.New("pi_peers_end").Parse(kPAGE_TEMPLATE_PEER_INFO_PEERS_END))
	serveTemplates["pi_foot"] = template.Must(template.New("pi_foot").Parse(kPAGE_TEMPLATE_FOOT_PEER_INFO))
}

func init() {
	prepareTemplates()
}
