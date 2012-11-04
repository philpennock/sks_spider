package sks_spider

import (
	"fmt"
	"io"
	"runtime"
)

func (spider *Spider) Diagnostic(out io.Writer) {
	fmt.Fprintf(out, "AddHost,BatchAddHost: %d, %d\n", len(spider.addHost), len(spider.batchAddHost))
	fmt.Fprintf(out, "Waitgroup: %#+v\n", spider.pending)
	n := runtime.NumGoroutine()
	spider.pendingDump <- out
	<-spider.doneDump
	fmt.Fprintf(out, "Go-routines: %d\n", n)
	fmt.Fprintf(out, "\n")
}
