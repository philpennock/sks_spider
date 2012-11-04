// Small utilities that may prove useful but have no better home
package sks_spider

import (
	"fmt"
	"io"
	"reflect"
)

func debugDumpMethods(out io.Writer, v interface{}) {
	t := reflect.TypeOf(v)
	max := t.NumMethod()
	for i := 0; i < max; i += 1 {
		m := t.Method(i)
		fmt.Fprintf(out, "M: %s\n", m.Name)
	}
}
