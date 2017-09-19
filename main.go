package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"
)

var webListenAddressFlag = flag.String("web.listen-address", ":10458", "address on which to expose metrics and logs")
var readFileFlag = flag.String("check.read-file", "", "name of a file to read under the mount point; if unspecified, default to readdir() of the mount point")
var timeoutFlag = flag.Int("check.timeout", 5, "seconds to wait until declaring a liveness check failed")
var fsTypesFlag = flag.String("check.fstypes", "nfs,nfs4,nfs3,cephfs,fuse.sshfs", "comma-separated file system types to include in the liveness check — pass the empty string to allow all")
var verboseFlag = flag.Bool("verbose", false, "print liveness check progress on standard error")

func verboseLog(str string, args ...interface{}) {
	if *verboseFlag {
		log.Printf(str, args...)
	}
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 2 {
		os.Exit(CheckViaSubprocess(flag.Args()[0], flag.Args()[1]))
	} else {
		ServeMetrics(*webListenAddressFlag,
			time.Duration(*timeoutFlag)*time.Second,
			strings.Split(*fsTypesFlag, ","),
			*readFileFlag)
	}

}
