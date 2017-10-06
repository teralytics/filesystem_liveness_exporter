// Copyright 2017 Teralytics.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		Server(*webListenAddressFlag,
			time.Duration(*timeoutFlag)*time.Second,
			strings.Split(*fsTypesFlag, ","),
			*readFileFlag)
	}
}
