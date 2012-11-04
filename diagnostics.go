package sks_spider

import (
	"fmt"
	"io"
	"runtime"
	"sync"
)

var diagnosticSpiderDump chan io.Writer
var diagnosticSpiderDone chan bool
var diagnosticSpiderKill chan bool
var diagnosticSpiderDummy bool
var diagnosticSpiderDummyLock sync.Mutex

func init() {
	diagnosticSpiderDump = make(chan io.Writer, 10)
	diagnosticSpiderDone = make(chan bool, 10)
	diagnosticSpiderKill = make(chan bool, 1)
}

func SpiderDiagnostics(out io.Writer) {
	diagnosticSpiderDump <- out
	<-diagnosticSpiderDone
}

func (spider *Spider) diagnosticDumpInRoutine(out io.Writer) {
	fmt.Fprintf(out, "AddHost,BatchAddHost: %d, %d\n", len(spider.addHost), len(spider.batchAddHost))
	fmt.Fprintf(out, "Waitgroup: %#+v\n", spider.pending)
	hostnames := make([]string, len(spider.pendingHosts))
	i := 0
	for h := range spider.pendingHosts {
		hostnames[i] = h
		i += 1
	}
	HostSort(hostnames)
	for _, h := range hostnames {
		count := spider.pendingHosts[h]
		if count != 0 {
			fmt.Fprintf(out, "\tWait: %3d  %s\n", count, h)
		}
	}
	n := runtime.NumGoroutine()
	fmt.Fprintf(out, "Go-routines: %d\n", n)
	fmt.Fprintf(out, "\n")
}

func KillDummySpiderForDiagnosticsChannel() {
	diagnosticSpiderDummyLock.Lock()
	defer diagnosticSpiderDummyLock.Unlock()
	if diagnosticSpiderDummy {
		diagnosticSpiderKill <- true
	}
}

func DummySpiderForDiagnosticsChannel() {
	okay := true
	func() {
		diagnosticSpiderDummyLock.Lock()
		defer diagnosticSpiderDummyLock.Unlock()
		if diagnosticSpiderDummy {
			okay = false
		} else {
			diagnosticSpiderDummy = true
		}
	}()
	if !okay {
		return
	}
	for {
		select {
		case <-diagnosticSpiderDump:
			diagnosticSpiderDone <- true
		case <-diagnosticSpiderKill:
			return
		}
	}
}
