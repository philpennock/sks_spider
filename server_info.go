/*
   Copyright 2009-2013,2017 Phil Pennock

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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	// "go-html-transform" -- crashes parsing SKS output
	// ehtml "exp/html" -- handles it, no xpath

	htmlp "github.com/sergioangulo/gokogiri/html"
	xml "github.com/sergioangulo/gokogiri/xml"
)

type SksNode struct {
	// Be sure that types of Exported fields are loadable from JSON!
	Hostname       string
	Port           int
	initialised    bool
	uriRel         string
	uri            string
	Status         string
	ServerHeader   string
	ViaHeader      string
	Settings       map[string]string
	GossipPeers    map[string]string
	GossipPeerList []string
	MailsyncPeers  []string
	Version        string
	Software       string
	Keycount       int
	pageContent    *htmlp.HtmlDocument
	pageJson       map[string]interface{}
	analyzeError   error

	// And these are populated when converted into a HostMap
	AnalyzeError string
	IpList       []string
	Aliases      []string
	Distance     int
}

var initHTTPOnce sync.Once
var ourTransport *http.Transport
var ourHTTPClient *http.Client

func getHTTPClient() *http.Client {
	initHTTPOnce.Do(func() {
		ourTransport = &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: *flHttpFetchTimeout,
		}
		ourHTTPClient = &http.Client{Transport: ourTransport}
	})
	return ourHTTPClient
}

func (sn *SksNode) Dump(out io.Writer) {
	fmt.Fprintf(out, "NODE: %s %d <%s>\n\tServer: %s\n\tVia: %s\n",
		sn.Hostname, sn.Port, sn.uri, sn.ServerHeader, sn.ViaHeader)
	fmt.Fprintf(out, "\tSoftware: %s\tVersion: %s\n\tKeycount: %d\n",
		sn.Software, sn.Version, sn.Keycount)
	for _, mail := range sn.MailsyncPeers {
		fmt.Fprintf(out, "\tMailsync to: %s\n", mail)
	}
	for k, v := range sn.Settings {
		fmt.Fprintf(out, "\tS: \"%s\" = \"%s\"\n", k, v)
	}
	for k, v := range sn.GossipPeers {
		fmt.Fprintf(out, "\tP: %s %s\n", k, v)
	}
	if sn.pageContent != nil {
		fmt.Fprintf(out, "\t%+v\n", sn.pageContent)
	} else {
		fmt.Fprint(out, "\tno page content\n")
	}
}

func (sn *SksNode) Normalize() bool {
	if sn.Hostname == "" {
		return false
	}
	if sn.initialised {
		return true
	}
	if sn.Port == 0 {
		sn.Port = *flSksPortHkp
	}
	if sn.Distance == 0 {
		// Will be overriden from the spider later
		sn.Distance = -1
	}
	sn.uriRel = "/pks/lookup?op=stats&options=mr"
	sn.uri = fmt.Sprintf("http://%s:%d%s", sn.Hostname, sn.Port, sn.uriRel)
	sn.initialised = true
	return true
}

// Dump the large content, let garbage collection reclaim space
func (sn *SksNode) Minimize() {
	if sn.pageContent != nil {
		sn.pageContent.Free()
		sn.pageContent = nil
	}
	// TODO: work out how to Free() the unstructured JSON object
	//	if sn.pageJson != nil {
	//		sn.pageJson.Free()
	//		sn.pageJson = nil
	//	}
}

func (sn *SksNode) Fetch() error {
	sn.Normalize()

	req, err := http.NewRequest("GET", sn.uri, nil)
	if err != nil {
		return err
	}
	// allow more time for whole context than for HTTP headers, hanoi stacking
	ctx, cancel := context.WithTimeout(context.Background(), *flHttpFetchTimeout+2*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", "sks_peers/0.2 (SKS mesh spidering)")
	cl := getHTTPClient()

	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	sn.Status = resp.Status
	Log.Printf("[%s] Response status: %s", sn.Hostname, sn.Status)
	sn.ServerHeader = resp.Header.Get("Server")
	sn.ViaHeader = resp.Header.Get("Via")
	//doc, err := ehtml.Parse(resp.Body)
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// try unmarshaling as JSON first
	var foo map[string]interface{}
	err = json.Unmarshal([]byte(buf), &foo)
	if err == nil {
		sn.pageJson = foo
		return nil
	}
	// otherwise assume it's an SKS-style HTML page
	doc, err := htmlp.Parse(buf, htmlp.DefaultEncodingBytes, nil, htmlp.DefaultParseOption, htmlp.DefaultEncodingBytes)
	if err != nil {
		return err
	}
	sn.pageContent = doc
	return nil
}

func (sn *SksNode) tableFollowing(search string) (table *xml.Node, err error) {
	if strings.ContainsRune(search, '"') {
		panic(fmt.Sprintf("Malformed search pattern {{{%s}}}", search))
	}
	res, err := sn.pageContent.Root().Search(fmt.Sprintf(`//*[text()="%s"]`, search))
	if err != nil {
		return nil, err
	}
	if len(res) < 1 {
		return nil, fmt.Errorf("Failed to find search text \"%s\"", search)
	}
	// NB: This only works for siblings, doesn't chase up, like python html5lib's .getnext() [lxml treebuilder]
	s := res[0]
	for s.Name() != "table" {
		s = s.NextSibling()
	}
	return &s, nil
}

func (sn *SksNode) plainRowsOf(search string) ([]string, error) {
	var rows []string
	table, err := sn.tableFollowing(search)
	if err != nil {
		return nil, err
	}
	//debugDumpMethods(*table)
	nodelist, err := (*table).Search(".//td")
	if err != nil {
		return nil, err
	}
	for i := range nodelist {
		text := strings.TrimSpace(nodelist[i].Content())
		rows = append(rows, text)
	}
	return rows, nil
}

func (sn *SksNode) dictFromPlainRows(search string) (map[string]string, error) {
	rows, err := sn.plainRowsOf(search)
	if err != nil {
		return nil, err
	}
	var dict = make(map[string]string)
	for i := range rows {
		elems := strings.SplitN(rows[i], " ", 2)
		dict[elems[0]] = elems[1]
	}
	return dict, err
}

func (sn *SksNode) kvdictFromTable(search string) (map[string]string, error) {
	table, err := sn.tableFollowing(search)
	if err != nil {
		return nil, err
	}
	nodelist, err := (*table).Search(".//tr")
	if err != nil {
		return nil, err
	}
	var dict = make(map[string]string)
	for i := range nodelist {
		columns, err := nodelist[i].Search(".//td")
		if err != nil {
			continue
		}
		key := strings.TrimSpace(columns[0].Content())
		value := strings.TrimSpace(columns[1].Content())
		key = strings.TrimRight(key, ":")
		dict[key] = value
	}
	return dict, nil
}

func (sn *SksNode) Analyze() {
	if !strings.HasPrefix(sn.Status, "200") {
		sn.Keycount = -2
		sn.analyzeError = fmt.Errorf("HTTP GET failure: %s", sn.Status)
		return
	}

	if sn.pageJson != nil {

		settings := make(map[string]string, 10)
		for key, val := range sn.pageJson {
			if valString, ok := val.(string); ok == true {
				settings[strings.Title(key)] = valString
			} else if valNum, ok := val.(float64); ok == true {
				settings[strings.Title(key)] = strconv.Itoa(int(valNum))
			}
		}
		sn.Settings = settings

		sn.Version = sn.Settings["Version"]
		sn.Software = sn.Settings["Software"]
		err := error(nil)
		sn.Keycount, err = strconv.Atoi(sn.Settings["Numkeys"])
		if err != nil {
			sn.Keycount, err = strconv.Atoi(sn.Settings["Total"])
			if err != nil {
				sn.Keycount = -1
			}
		}

		if peerArray, ok := sn.pageJson["peers"].([]interface{}); ok == true {
			sn.GossipPeers = make(map[string]string, len(peerArray))
			sn.GossipPeerList = make([]string, len(peerArray))
			for i, peer := range peerArray {
				if peerMap, ok := peer.(map[string]interface{}); ok == true {
					if reconAddr, ok := peerMap["reconAddr"].(string); ok == true {
						if strings.ContainsAny(reconAddr, ":") {
							// TODO: split this on the LAST colon, but only if there are no spaces (ipv6 parsing)
							reconAddr = strings.Replace(reconAddr, ":", " ", 1)
						}
						sn.GossipPeers[strings.Fields(reconAddr)[0]] = strings.Fields(reconAddr)[1]
						sn.GossipPeerList[i] = strings.Fields(reconAddr)[0]
					}
				}
			}
		}

	} else {

		if mailsync, err := sn.plainRowsOf("Outgoing Mailsync Peers"); err == nil {
			sn.MailsyncPeers = mailsync
		}
		if settings, err := sn.kvdictFromTable("Settings"); err == nil {
			sn.Settings = settings
		}
		sn.Version = sn.Settings["Version"]
		sn.Software = sn.Settings["Software"]
		if res, err := sn.pageContent.Root().Search(`//h2[text()="Statistics"]`); err == nil {
			content := res[0].NextSibling().Content()
			if strings.HasPrefix(content, "Total number of keys") {
				content = strings.TrimSpace(strings.SplitN(content, ":", 2)[1])
				sn.Keycount, err = strconv.Atoi(content)
				if err != nil {
					sn.Keycount = -1
				}
			}
		}

		if peers, err := sn.dictFromPlainRows("Gossip Peers"); err == nil {
			sn.GossipPeerList = make([]string, len(peers))
			var i = 0
			for k := range peers {
				sn.GossipPeerList[i] = k
				i += 1
			}

			for _, k := range sn.GossipPeerList {
				if strings.ContainsAny(peers[k], " \t") {
					peers[k] = strings.Fields(peers[k])[0]
				}
			}
			sn.GossipPeers = peers
		}

	}

	sn.Minimize()
}

func (sn *SksNode) Url() string {
	if sn.uri != "" {
		return sn.uri
	}
	// JSON reloaded
	return fmt.Sprintf("http://%s:%d/pks/lookup?op=stats&options=mr", sn.Hostname, sn.Port)
}

func NodeUrl(name string, sn *SksNode) string {
	if sn != nil {
		return sn.Url()
	}
	return fmt.Sprintf("http://%s:%d/pks/lookup?op=stats&options=mr", name, *flSksPortHkp)
}
