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

type metricsHandler struct {
	timeout     time.Duration
	fsTypes     []string
	optReadFile string
}

func (m *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fses := DiscoverFilesystems("/proc/mounts", m.fsTypes)
	fslist := []*FilesystemChecker{}
	for _, fs := range fses {
		fslist = append(fslist, &FilesystemChecker{fs})
	}
	waits := make(map[*FilesystemChecker]func() *LivenessCheck)
	for _, fs := range fslist {
		waits[fs] = fs.Check(m.timeout, m.optReadFile)
	}
	for _, fs := range fslist {
		check := waits[fs]()
		for _, elm := range []*metricsElement{
			&metricsElement{"vfs_filesystem_error", fs.mountpoint, boolToFloat(check.err)},
			&metricsElement{"vfs_filesystem_live", fs.mountpoint, boolToFloat(check.live)},
			&metricsElement{"vfs_filesystem_scan_duration_seconds", fs.mountpoint, check.duration},
		} {
			fmt.Fprintf(w, "%s", elm)
		}
	}
}

func Server(listenAddr string, collectTimeout time.Duration, fsTypes []string, optReadFile string) {
	log.Printf("Serving status and metrics on address %s", listenAddr)
	srv := &http.Server{
		Addr:           listenAddr,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   (5 * time.Second) + collectTimeout,
		MaxHeaderBytes: 4096,
	}
	http.Handle("/metrics", &metricsHandler{collectTimeout, fsTypes, optReadFile})
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
