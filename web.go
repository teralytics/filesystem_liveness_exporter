package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type metricsElement struct {
	Name       string
	Mountpoint string
	Value      float64
}

func (m metricsElement) String() string {
	mt := strings.Replace(
		strings.Replace(
			strings.Replace(m.Mountpoint, "\\", "\\\\", -1),
			"\n", "\\n", -1), "\"", "\\\"", -1)
	return fmt.Sprintf("%s {mountpoint=\"%s\"} %f\n", m.Name, mt, m.Value)
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func dumpMetrics(res []*CheckResult, w http.ResponseWriter, r *http.Request) {
	for _, item := range res {
		for _, elm := range []*metricsElement{
			&metricsElement{"vfs_filesystem_error", item.filesystem.mountpoint, boolToFloat(item.check.err)},
			&metricsElement{"vfs_filesystem_live", item.filesystem.mountpoint, boolToFloat(item.check.live)},
			&metricsElement{"vfs_filesystem_scan_duration_seconds", item.filesystem.mountpoint, item.check.duration},
		} {
			fmt.Fprintf(w, "%s", elm)
		}
	}
}

func metrics(timeout time.Duration, fsTypes []string, optReadFile string, w http.ResponseWriter, r *http.Request) {
	res := CollectMetrics(timeout, fsTypes, optReadFile)
	dumpMetrics(res, w, r)
}

func ServeMetrics(listenAddr string, collectTimeout time.Duration, fsTypes []string, optReadFile string) {
	log.Printf("Serving status and metrics on address %s", listenAddr)
	srv := &http.Server{
		Addr:           listenAddr,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   (5 * time.Second) + collectTimeout,
		MaxHeaderBytes: 4096,
	}
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics(collectTimeout, fsTypes, optReadFile, w, r)
	})
	//http.HandleFunc("/quitquitquit", func(http.ResponseWriter, *http.Request) { os.Exit(0) })
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
  <head><title>File system liveness exporter</title></head>
  <body>
    <H1>File system liveness exporter</H1>
    <p><a href="/metrics">Metrics</a></p>
  </body>
</html>`))
	})
	srv.ListenAndServe()
}
