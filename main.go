package sks_spider

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	flSpiderStartHost    = flag.String("spider-start-host", "sks-peer.spodhuis.org", "Host to query to start things rolling")
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
	flJsonDump           = flag.String("json-dump", "", "File to dump JSON of spidered hosts to")
	//flJsonLoad           = flag.String("json-load", "", "File to load JSON hosts from instead of spidering")
	flJsonLoad = flag.String("json-load", "dump-hosts-2012-11-04.json", "File to load JSON hosts from instead of spidering")
)

var serverHeadersNative = map[string]bool{
	"sks_www": true,
	"gnuks":   true,
}

// People put dumb things in their membership files
var blacklistedQueryHosts = []string{
	"localhost",
	"127.0.0.1",
	"::1",
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

func statusPeriodicDump(spider *Spider, stop <-chan bool) {
	for {
		select {
		case <-time.After(time.Second * 10):
			spider.Diagnostic(os.Stdout)
		case <-stop:
			break
		}
	}
}

// TODO: switch to a straight map, drop the spider gunk
var (
	currentMesh     *Spider
	currentMeshLock sync.Mutex
)

func GetCurrentMesh() *Spider {
	currentMeshLock.Lock()
	defer currentMeshLock.Unlock()
	return currentMesh
}

func SetCurrentMesh(spider *Spider) {
	currentMeshLock.Lock()
	defer currentMeshLock.Unlock()
	currentMesh = spider
}

func respiderPeriodically() {
	for {
		var delay time.Duration = time.Duration(*flScanIntervalSecs) * time.Second
		if *flScanIntervalJitter > 0 {
			jitter := rand.Int63n(int64(*flScanIntervalJitter) * int64(time.Second))
			jitter -= int64(*flScanIntervalJitter) * int64(time.Second) / 2
			delay += time.Duration(jitter)
		}
		minDelay := time.Minute * 30
		if delay < minDelay {
			Log.Printf("respider period too low, capping %d up to %d", delay, minDelay)
			delay = minDelay
		}
		Log.Printf("Sleeping %s before next respider", delay)
		time.Sleep(delay)
		Log.Printf("Awoken!  Time to spider.")
		spider := StartSpider()
		spider.AddHost(*flSpiderStartHost)
		spider.Wait()
		SetCurrentMesh(spider)
		runtime.GC()
	}
}

func Main() {
	flag.Parse()

	if *flScanIntervalJitter < 0 {
		fmt.Fprintf(os.Stderr, "Bad jitter, must be >= 0 [got: %d]\n", *flScanIntervalJitter)
		os.Exit(1)
	}

	setupLogging()
	Log.Printf("started")

	var spider *Spider
	var err error

	if *flJsonLoad != "" {
		Log.Printf("Loading hosts from \"%s\" instead of spidering", *flJsonLoad)
		spider, err = LoadJSONFromFile(*flJsonLoad)
		if err != nil {
			Log.Fatalf("Failed to load JSON from \"%s\": %s", *flJsonLoad, err)
		}
	} else {
		spider = StartSpider()
		spider.AddHost(*flSpiderStartHost)
		//stop := make(chan bool)
		//go statusPeriodicDump(spider, stop)
		spider.Wait()
		//stop <- true
		Log.Printf("Spidering complete")
		if *flJsonDump != "" {
			err = spider.DumpJSONToFile(*flJsonDump)
			if err != nil {
				Log.Printf("Error saving JSON to \"%s\": %s", *flJsonDump, err)
				// continue anyway
			}
		}
	}

	SetCurrentMesh(spider)
	runtime.GC()

	if *flJsonLoad == "" {
		go respiderPeriodically()
	}

	fmt.Printf("\nSPIDER: %#+v\n", spider)
}
