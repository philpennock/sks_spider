package sks_spider

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect" // debugDumpMethods() below
	"strconv"
	"strings"
)

// "go-html-transform" -- crashes parsing SKS output
// ehtml "exp/html" -- handles it, no xpath

import (
	htmlp "github.com/moovweb/gokogiri/html"
	xml "github.com/moovweb/gokogiri/xml"
)

// Why do XML/HTML-tree libraries come with such poor docs?  Oh right, because HTML is so barely-specified
func debugDumpMethods(v interface{}) {
	t := reflect.TypeOf(v)
	max := t.NumMethod()
	for i := 0; i < max; i += 1 {
		m := t.Method(i)
		fmt.Printf("M: %s\n", m.Name)
	}
}

type SksNode struct {
	Hostname      string
	Port          int
	initialised   bool
	uriRel        string
	uri           string
	ServerHeader  string
	ViaHeader     string
	Settings      map[string]string
	GossipPeers		map[string]string
	MailsyncPeers []string
	Version       string
	Software      string
	Keycount      int
	pageContent   *htmlp.HtmlDocument
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
	sn.uriRel = "/pks/lookup?op=stats"
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
}

func (sn *SksNode) Fetch() error {
	sn.Normalize()
	req, err := http.NewRequest("GET", sn.uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "sks_peers/0.2 (SKS mesh spidering)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	Log.Printf("Response status: %s", resp.Status)
	sn.ServerHeader = resp.Header.Get("Server")
	sn.ViaHeader = resp.Header.Get("Via")
	//doc, err := ehtml.Parse(resp.Body)
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
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
		for k, v := range peers {
			if strings.ContainsAny(v, " \t") {
				peers[k] = strings.Fields(v)[0]
			}
		}
		sn.GossipPeers = peers
	}

	sn.Minimize()
}
