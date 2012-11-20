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
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type GraphvizAttributes map[string]interface{}

func (ga GraphvizAttributes) String() string {
	// http://www.graphviz.org/doc/info/lang.html
	if len(ga) == 0 {
		return ""
	}
	buf := new(bytes.Buffer)
	for k, v := range ga {
		var encodedV string
		switch v.(type) {
		case int:
			encodedV = strconv.Itoa(v.(int))
		case bool:
			encodedV = strconv.FormatBool(v.(bool))
		default:
			encodedV = fmt.Sprintf("\"%s\"", v)
		}
		buf.WriteString(k)
		buf.WriteRune('=')
		buf.WriteString(encodedV)
		buf.WriteString(", ")
	}
	buf.Truncate(buf.Len() - 2)
	return buf.String()
}

func apiGraphDot(w http.ResponseWriter, req *http.Request) {
	persisted := GetCurrentPersisted()
	if persisted == nil {
		http.Error(w, "Still awaiting data collection", http.StatusServiceUnavailable)
		return
	}
	timestamp := time.Now().UTC().Format("20060102_150405") + "Z"
	filename := fmt.Sprintf("sks-peers-%s.dot", timestamp)
	w.Header().Set("Content-Type", "text/x-graphviz; charset=UTF-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	if req.Method == "HEAD" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// If we add a bi-directional link A<->B, then we put a key "B:A" into this
	// map, so that we don't emit a second bi-directional from the other side.
	shown := make(map[string]bool, len(persisted.Sorted) * len(persisted.Sorted))

	// There's also "concentrate" but it's not fully supported and I'm not sure
	// about what is meant by "causes partially parallel edges to share part of
	// their paths".
	// Semantically, we'll have to hope that dir=both is more than just decoration.

	// We'll emit constraint=false because it's a mesh of peers, none more
	// important than another.  Not even the seed we happened to use.

	fmt.Fprintf(w, "digraph sks {\n")
	for _, hostname := range persisted.Sorted {
		attributes := make(GraphvizAttributes)
		node := persisted.HostMap[hostname]
		attributes["depth"] = node.Distance
		if node.AnalyzeError != "" {
			attributes["error"] = node.AnalyzeError
		} else {
			attributes["software"] = node.Software
			attributes["version"] = node.Version
			attributes["keycount"] = node.Keycount
		}
		for n, ip := range node.IpList {
			attributes[fmt.Sprintf("ip%d", n)] = ip
		}
		fmt.Fprintf(w, "\t\"%s\" [%s];\n", hostname, attributes)
	}
	var directionality string
	for _, hostname := range persisted.Sorted {
		for peername := range persisted.Graph.Outbound(hostname) {
			backwards := fmt.Sprintf("%s:%s", peername, hostname)
			if shown[backwards] {
				continue
			}
			if persisted.Graph.ExistsLink(peername, hostname) {
				directionality = " dir=both"
				shown[backwards] = true
			} else {
				directionality = ""
			}
			fmt.Fprintf(w, "\t\"%s\" -> \"%s\" [constraint=false%s];\n", hostname, peername, directionality)
		}
	}
	fmt.Fprintf(w, "}\n")

}
