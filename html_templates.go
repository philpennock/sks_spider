/*
   Copyright 2009-2013,2018 Phil Pennock

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
	"fmt"
	"html/template"
)

var serveTemplates map[string]*template.Template

func genNamespace() map[string]interface{} {
	ns := make(map[string]interface{}, 50)
	ns["Maintainer"] = *flMaintEmail
	ns["StartHost"] = *flSpiderStartHost
	ns["MyStylesheet"] = *flMyStylesheet
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
  <link rel="stylesheet" href="{{.MyStylesheet}}" type="text/css" charset="utf-8">
  <link rel="shortcut icon" href="%s" type="image/x-icon">
`, kHTML_FAVICON)

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
  <title>SKS Peer Mesh</title>
 </head>
 <body>
  <h1>SKS Peer Mesh</h1>
{{.Warning}}
{{.Scanning_active}}
  <div class="explain">
   Entries at depth 1 are direct peers of <span class="hostname">{{.StartHost}}</span>.
   Others are seen by spidering the peers.
  </div>
  <table class="sks peertable">
   <thead><tr><th>Host</th><th>Info</th><th>IP</th><th>Geo</th><th>Mutual</th><th>Version</th><th>Keys</th><th>Distance</th><th>WebServer</th><th>Proxy/via</th></tr></thead>
   <tbody>
`

	kPAGE_TEMPLATE_FOOT := `
   <caption>SKS has {{.Peer_count}} peers of {{.Mesh_count}} visible</caption>
  </table>
  <div class="lastupdate">Last scan completed at: {{.LastScanTime}}</div>
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
    <td class="exception" colspan="5">Error: {{.Error}}</td>
    <td class="peer_distance">{{.Distance}}</td>
	<td colspan="2"></td>
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
   <tr><td rowspan="{{.Mailsync_count}}">Mailsync</td>{{$need_tr := false}}
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
   <caption>Peers of <span class="hostname">{{.Peername}}</span></caption>
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
