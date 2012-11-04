package sks_spider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const SERVE_PREFIX = "/sks-peers"

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
}

func apiPeersPage(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	fmt.Fprintf(w, "A peers page, perchance?\n")
}

func apiPeerInfoPage(w http.ResponseWriter, req *http.Request) {
}

func apiIpValidPage(w http.ResponseWriter, req *http.Request) {
}

func apiIpValidStatsPage(w http.ResponseWriter, req *http.Request) {
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
		if len(hosts) == 0 {
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
