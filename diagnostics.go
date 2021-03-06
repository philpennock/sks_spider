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
	diagnosticSpiderDump = make(chan io.Writer)
	diagnosticSpiderDone = make(chan bool)
	diagnosticSpiderKill = make(chan bool)
}

func SpiderDiagnostics(out io.Writer) {
	diagnosticSpiderDump <- out
	<-diagnosticSpiderDone
}

// spiderMainLoop(), which calls us, will signal on diagnosticSpiderDone when done
func (spider *Spider) diagnosticDumpInRoutine(out io.Writer) {
	fmt.Fprintf(out, "BatchAddHost: %d / %d\n", len(spider.batchAddHost), cap(spider.batchAddHost))
	// We can no longer print %#+v the spider.pending sync.WaitGroup because it's marked noCopy and
	// the print counts as a copy for the purposes of `go vet`
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
			diagnosticSpiderDummyLock.Lock()
			defer diagnosticSpiderDummyLock.Unlock()
			diagnosticSpiderDummy = false
			return
		}
	}
}
