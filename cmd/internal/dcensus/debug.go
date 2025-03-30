package dcensus

import (
	"fmt"
	"net/http"
	"os"

	hpprof "net/http/pprof"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/tetsuo/fortune/internal/memory"
	"go.opencensus.io/zpages"
	"go.uber.org/zap"
)

func debugHomeHandler(versionID, instanceID, name, commitHash, commitDate string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
<html>
<style>
:root{
  background:#fff;
  font-family:sans-serif;
}
</style>
<body>
	<h2>%s/frontend</h2>
	<code>
		instance=%s
		version=%s
		commit=%s
		date=%s
	</code>
	<ul>
	<li><a href="/tracez">Traces</a></li>
	<li><a href="/statsz">Metrics</a></li>
	<li><a href="/_debug/info">Resources</a></li>
		<li><a href="/debug/pprof">Pprof</a></li>
	</ul>
</body>
</html>
`, name, instanceID, versionID, commitHash, commitDate)
	}
}

type debugServer struct {
	versionID  string
	instanceID string
}

func (s *debugServer) debugInfoHandler(w http.ResponseWriter, r *http.Request) {
	gm := memory.ReadRuntimeStats()
	pm, err := memory.ReadProcessStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading process stats: %v", err), http.StatusInternalServerError)
		return
	}
	sm, err := memory.ReadSystemStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading system stats: %v", err), http.StatusInternalServerError)
		return
	}
	row := func(a, b string) {
		fmt.Fprintf(w, "<tr><td>%s</td> <td>%s</td></tr>\n", a, b)
	}
	memrow := func(s string, m uint64) {
		fmt.Fprintf(w, "<tr><td>%s</td> <td align='right'>%s</td></tr>\n", s, memory.Format(m))
	}
	countrow := func(s string, m int) {
		fmt.Fprintf(w, "<tr><td>%s</td> <td align='right'>%d</td></tr>\n", s, m)
	}
	fmt.Fprintf(w, "<html><body style='font-family: sans-serif'>\n")
	fmt.Fprintf(w, "<table border=1>\n")
	row("Service", os.Getenv("K_SERVICE"))
	row("Config", os.Getenv("K_CONFIGURATION"))
	row("Version", s.versionID)
	row("Instance", s.instanceID)
	fmt.Fprintf(w, "</table>\n<br>\n")
	fmt.Fprintf(w, "<table border=1>\n")
	memrow("Go Sys", gm.Sys)
	memrow("Go GCSys", gm.GCSys)
	memrow("Go Alloc", gm.Alloc)
	memrow("Go TotalAlloc", gm.TotalAlloc)
	countrow("Go NumGC", int(gm.NumGC))
	countrow("Go Mallocs", int(gm.Mallocs))
	countrow("Go Frees", int(gm.Frees))
	memrow("Go HeapAlloc", gm.HeapAlloc)
	memrow("Go HeapIdle", gm.HeapIdle)
	memrow("Go HeapInuse", gm.HeapInuse)
	memrow("Go HeapReleased", gm.HeapReleased)
	memrow("Go HeapSys", gm.HeapSys)
	countrow("Go HeapObjects", int(gm.HeapObjects))
	memrow("Process VSize", pm.VSize)
	memrow("Process RSS", pm.RSS)
	memrow("System Total", sm.Total)
	memrow("System Used", sm.Used)
	cm, err := memory.ReadCgroupStats()
	if err != nil {
		row("CGroup Stats", "unavailable")
	} else {
		for k, v := range cm {
			memrow("CGroup "+k, v)
		}
	}
	fmt.Fprintf(w, "</table>\n")
	fmt.Fprintf(w, "</body></html>\n")
}

// ServeDebug serves the internal debug server.
func ServeDebug(pe *prometheus.Exporter, pprofEnabled bool, versionID, instanceID, name, commit, date string) (http.Handler, error) {
	mux := http.NewServeMux()
	zpages.Handle(mux, "/")
	if pe != nil {
		mux.Handle("/statsz", pe)
		// https://github.com/GoogleCloudPlatform/prometheus-engine/blob/v0.4.1/doc/api.md#scrapeendpoint
		mux.Handle("/metrics", pe)
	}
	mux.HandleFunc("/", debugHomeHandler(versionID, instanceID, name, commit, date))
	if pprofEnabled {
		zap.S().Info("enabling pprof handler")
		mux.Handle("/debug/pprof/", http.HandlerFunc(hpprof.Index))
		mux.Handle("/_debug/pprof/cmdline", http.HandlerFunc(hpprof.Cmdline))
		mux.Handle("/_debug/pprof/profile", http.HandlerFunc(hpprof.Profile))
		mux.Handle("/_debug/pprof/symbol", http.HandlerFunc(hpprof.Symbol))
		mux.Handle("/_debug/pprof/trace", http.HandlerFunc(hpprof.Trace))
	}
	s := new(debugServer)
	s.versionID = versionID
	s.instanceID = instanceID
	mux.Handle("/_debug/info", http.HandlerFunc(s.debugInfoHandler))
	return mux, nil
}
