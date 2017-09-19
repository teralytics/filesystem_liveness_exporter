package main

import (
	"io/ioutil"
	"log"
	"os"
	"syscall"
)

func CheckViaSubprocess(path string) (exitstatus int) {
	_, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsPermission(err) {
			// Permission denied indicates the file system is alive.
			exitstatus = 0
		} else if serr, ok := err.(*os.SyscallError); ok && (serr.Err == syscall.ENOTDIR) {
			// Is not a directory indicates the file system is alive
			exitstatus = 0
		} else {
			log.Printf("Error: cannot read %s: (%T) %s", path, err, err)
			exitstatus = 4
		}
	}
	return
}
