package main

import (
	"io/ioutil"
	"log"
	"os"
	"syscall"
)

func CheckViaSubprocess(mode, path string) (exitstatus int) {
	var err error
	if mode == "readdir" {
		_, err = ioutil.ReadDir(path)
	} else if mode == "read" {
		_, err = ioutil.ReadFile(path)
	} else {
		log.Printf("Error: internal checker mode accepts two arguments: <read | readdir> <path>")
		exitstatus = 64
	}
	if err != nil {
		if os.IsPermission(err) {
			// Permission denied indicates the file system is alive.
			// We return zero, as if the check had passed.
			exitstatus = 0
		} else if serr, ok := err.(*os.SyscallError); ok && (serr.Err == syscall.ENOTDIR) {
			// Is not a directory indicates the file system is alive
			// We return zero, as if the check had passed.
			exitstatus = 0
		} else {
			log.Printf("Checker: cannot %s() %s: (%T) %s", mode, path, err, err)
			exitstatus = 4
		}
	}
	return
}
