package sks_spider

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

const SERVE_PREFIX = "/sks-peers"

const (
	kHTML_CSS_STYLESHEET = "/style/spodhuis-sks.css"
	kHTML_FAVICON        = "/favicon.ico"
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
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	fmt.Fprintf(w, "A peers page, perchance?\n")
	//TODO: write
	hostnames := GetCurrentHostlist()
	for _, h := range hostnames {
		fmt.Fprintf(w, "H: %s\n", h)
	}
	fmt.Fprintf(w, ".\n")
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

	if len(persisted.Sorted) == 0 {
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

	peer_list := make([]string, len(node.GossipPeerList))
	copy(peer_list, node.GossipPeerList)
	HostSort(peer_list)

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
		if _, ok := persisted.HostMap[other]; !ok {
			// peer not successfully polled
			attributes["In"] = "?"
			attributes["Common"] = "?"
		} else {
			attributes["In"] = in
			attributes["Common"] = common
		}
		Log.Printf("Showing peer info %s -> %s", peer, other)
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
	fmt.Fprintf(w, "ipValid: %#+v\n", req)
	//TODO: write
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
    <td class="hostname"{{.Rowspan}}>{{.Html_link}}{{.Host_aliases_text}}</td>
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
