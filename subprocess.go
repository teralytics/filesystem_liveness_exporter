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
		panic(mode)
	}
	if err != nil {
		if os.IsPermission(err) {
			// Permission denied indicates the file system is alive.
			exitstatus = 0
		} else if serr, ok := err.(*os.SyscallError); ok && (serr.Err == syscall.ENOTDIR) {
			// Is not a directory indicates the file system is alive
			exitstatus = 0
		} else {
			log.Printf("Checker: cannot %s() %s: (%T) %s", mode, path, err, err)
			exitstatus = 4
		}
	}
	return
}
