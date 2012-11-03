package sks_spider

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	flMaintEmail         = flag.String("maint-email", "webmaster@spodhuis.org", "Email address of local maintainer")
	flSksMembershipFile  = flag.String("sks-membership-file", "/var/sks/membership", "SKS Membership file")
	flSksPortRecon       = flag.Int("sks-port-recon", 11370, "Default SKS recon port")
	flSksPortHkp         = flag.Int("sks-port-hkp", 11371, "Default SKS HKP port")
	flTimeoutStatsFetch  = flag.Int("timeout-stats-fetch", 30, "Timeout for fetching stats from a remote server")
	flCountriesZone      = flag.String("countries-zone", "zz.countries.nerd.dk.", "DNS zone for determining IP locations")
	flKeysSanityMin      = flag.Int("keys-sanity-min", 3100000, "Minimum number of keys that's sane, or we're broken")
	flKeysDailyJitter    = flag.Int("keys-daily-jitter", 500, "Max daily jitter in key count")
	flScanIntervalSecs   = flag.Int("scan-interval", 3600*8, "How often to trigger a scan")
	flScanIntervalJitter = flag.Int("scan-interval-jitter", 120, "Jitter in scan interval")
	flLogFile            = flag.String("log-file", "sksdaemon.log", "Where to write logfiles")
)

var serverHeadersNative = map[string]bool{
	"sks_www": true,
	"gnuks":   true,
}

var Log *log.Logger

func setupLogging() {
	fh, err := os.OpenFile(*flLogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to open logfile \"%s\": %s\n", *flLogFile, err)
		os.Exit(1)
	}
	Log = log.New(fh, "", log.LstdFlags|log.Lshortfile)
}

func Main() {
	flag.Parse()
	setupLogging()
	Log.Printf("started")

	node := &SksNode{Hostname: "sks.spodhuis.org"}
	node.Fetch()
	node.Analyze()
	node.Dump(os.Stdout)
}
