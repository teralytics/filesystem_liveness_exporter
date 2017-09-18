package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var webListenAddressFlag = flag.String("web.listen-address", ":10458", "address on which to expose metrics and logs")
var timeoutFlag = flag.Int("timeout", 5, "seconds to wait until declaring a liveness check failed")
var fsTypesFlag = flag.String("fstypes", "nfs,nfs4,nfs3,cephfs,fuse.sshfs", "comma-separated file system types to include in the liveness check — pass the empty string to allow all")

type Filesystem struct {
	device     string
	mountpoint string
	fstype     string
}

type LivenessCheck struct {
	skip     bool
	live     bool
	duration float64
}

type CheckResult struct {
	filesystem *Filesystem
	check      *LivenessCheck
}

func (x *Filesystem) Check(timeout time.Duration) func() *LivenessCheck {
	doneChan := make(chan *LivenessCheck)

	go func(c chan *LivenessCheck) {
		start := time.Now()
		// 		if x.mountpoint == "/" {
		// 			time.Sleep(950 * time.Millisecond)
		// 		}
		_, err := ioutil.ReadDir(x.mountpoint)
		end := time.Now()
		lc := &LivenessCheck{
			duration: float64(end.Sub(start)) / 1000000000,
		}
		if err != nil {
			if os.IsPermission(err) {
				// Permission denied indicates the file system is alive.
				lc.live = true
			} else if serr, ok := err.(*os.SyscallError); ok && (serr.Err == syscall.ENOTDIR) {
				// Is not a directory indicates the file system is alive
				lc.live = true
			} else {
				log.Printf("error: cannot read %s: (%T) %s", x.mountpoint, err, err)
				lc.skip = true
			}
		} else {
			lc.live = true
		}
		c <- lc
		close(c)
	}(doneChan)

	return func() *LivenessCheck {
		select {
		case y := <-doneChan:
			return y
		case <-time.After(timeout):
			return &LivenessCheck{}
		}
	}
}

// unquoteKernelMount takes a string quoted in the form of
// This\040is\040a\040mountpoint\040with\040spaces and
// undoes the quoting of octal characters.  The kernel
// uses that quoting to protect shell programs parsing
// /proc/mounts from erroneously parsing device files and
// mount points that have special characters in them, and
// to disambiguate them.
//
// It returns the unquoted path as in
// This is a mountpoint with spaces
func unquoteKernelMount(quoted string) (string, error) {
	unquoted := ""
	for i := 0; i < len(quoted); i++ {
		if quoted[i] == "\\"[0] {
			if len(quoted) < i+4 {
				return "", fmt.Errorf("string %s is an invalid path", quoted)
			}
			x, err := strconv.ParseUint(quoted[i+1:i+4], 8, 8)
			if err != nil {
				return "", err
			}
			unquoted = unquoted + fmt.Sprintf("%c", x)
			i = i + 3
		} else {
			unquoted = unquoted + string(quoted[i])
		}
	}
	return unquoted, nil
}

func scanFilesystems(mountsFile string, allowedFsTypes []string) []*Filesystem {
	f, err := os.Open(mountsFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := bufio.NewScanner(f)
	fses := []*Filesystem{}
	for r.Scan() {
		sp := strings.Split(r.Text(), " ")
		if len(sp) < 3 {
			continue
		}
		correctType := len(allowedFsTypes) == 0 || (len(allowedFsTypes) == 1 && allowedFsTypes[0] == "")
		for _, fsType := range allowedFsTypes {
			if fsType == sp[2] {
				correctType = true
				break
			}
		}
		if !correctType {
			continue
		}
		unquotedDevice, err := unquoteKernelMount(sp[0])
		if err != nil {
			panic(err)
		}
		unquotedMountpoint, err := unquoteKernelMount(sp[1])
		if err != nil {
			panic(err)
		}
		fses = append(fses, &Filesystem{
			device:     unquotedDevice,
			mountpoint: unquotedMountpoint,
			fstype:     sp[2],
		})
	}
	return fses
}

func collectMetrics() []*CheckResult {
	fses := scanFilesystems("/proc/mounts", strings.Split(*fsTypesFlag, ","))
	waits := make(map[*Filesystem]func() *LivenessCheck)
	for _, fs := range fses {
		waits[fs] = fs.Check(time.Duration(*timeoutFlag) * time.Second)
	}
	res := []*CheckResult{}
	for _, fs := range fses {
		res = append(res, &CheckResult{fs, waits[fs]()})
	}
	return res
}

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

func dumpMetrics(res []*CheckResult, w http.ResponseWriter, r *http.Request) {
	for _, item := range res {
		if item.check.skip {
			continue
		}
		m := metricsElement{"vfs_filesystem_live", item.filesystem.mountpoint, 0.0}
		if item.check.live {
			m.Value = 1.0
		}
		fmt.Fprintf(w, "%s", m)
		if item.check.live {
			n := metricsElement{"vfs_filesystem_scan_duration_seconds", item.filesystem.mountpoint, item.check.duration}
			fmt.Fprintf(w, "%s", n)
		}
	}
}

func metrics(w http.ResponseWriter, r *http.Request) {
	res := collectMetrics()
	dumpMetrics(res, w, r)
}

func main() {
	flag.Parse()
	log.Printf("Serving status and metrics on address %s", *webListenAddressFlag)
	http.HandleFunc("/metrics", metrics)
	//	http.HandleFunc("/quitquitquit", func(http.ResponseWriter, *http.Request) { os.Exit(0) } )
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
  <head><title>File system liveness exporter</title></head>
  <body>
    <H1>File system liveness exporter</H1>
    <p><a href="/metrics">Metrics</a></p>
  </body>
</html>`))
	})
	http.ListenAndServe(*webListenAddressFlag, nil)
}
