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
	"net/http"
)

import (
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/vgimg"
)

// See https://code.google.com/p/plotinum/wiki/Examples

func apiPlotServersHistogram(w http.ResponseWriter, req *http.Request) {
	persisted := GetCurrentPersisted()
	if persisted == nil {
		http.Error(w, "Still waiting for data collection", http.StatusServiceUnavailable)
		return
	}
	values := make(plotter.Values, len(persisted.Sorted))
	vindex := -1
	for _, name := range persisted.Sorted {
		node := persisted.HostMap[name]
		if node.Keycount <= 1 {
			continue
		}
		vindex++
		values[vindex] = float64(node.Keycount)
	}
	if vindex < 0 {
		http.Error(w, "No servers with keycounts", http.StatusInternalServerError)
		return
	}
	values = values[:vindex+1]

	p, err := plot.New()
	if err != nil {
		Log.Printf("plot.New() failed: %s", err)
		http.Error(w, "Plot creation glitch", http.StatusInternalServerError)
		return
	}

	p.Title.Text = "SKS Keyserver keycounts"
	h := plotter.NewHist(values, 40 /*XXX*/)
	p.Add(h)

	width, height := vg.Inches(6), vg.Inches(5) // XXX
	canvas := vgimg.PngCanvas{vgimg.New(width, height)}
	p.Draw(plot.MakeDrawArea(canvas))

	w.Header().Set("Content-Type", "image/png")
	canvas.WriteTo(w)
}
