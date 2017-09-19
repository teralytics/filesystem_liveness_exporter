package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// FilesystemChecker holds the data of the filesystem to be checked
// and loads it with the functionality necessary to run a check.
type FilesystemChecker struct {
	*Filesystem
}

// LivenessCheck is the result that Check() returns.
type LivenessCheck struct {
	err      bool
	live     bool
	duration float64
}

// Check runs an asynchronous check on a FilesystemChecker.
func (x *FilesystemChecker) Check(timeout time.Duration, optReadFile string) func() *LivenessCheck {
	doneChan := make(chan *LivenessCheck)

	start := time.Now()
	lc := &LivenessCheck{}
	myself, err := executable()
	if err != nil {
		log.Printf("Error: cannot find myself: (%T) %s", err, err)
		lc.err = true
		lc.duration = float64(time.Now().Sub(start)) / 1000000000
		return func() *LivenessCheck { return lc }
	}

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	var cmd *exec.Cmd
	if optReadFile != "" {
		cmd = exec.CommandContext(ctx, myself, "read", filepath.Join(x.mountpoint, optReadFile))
	} else {
		cmd = exec.CommandContext(ctx, myself, "readdir", x.mountpoint)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		verboseLog("Starting liveness check of %s", x.mountpoint)
		err = cmd.Run()
		verboseLog("Ended liveness check of %s", x.mountpoint)
		lc.duration = float64(time.Now().Sub(start)) / 1000000000
		if err != nil {
			if eerr, ok := err.(*exec.ExitError); ok {
				if !eerr.Sys().(syscall.WaitStatus).Signaled() {
					log.Printf("Error: checker subprocess for %s failed: (%T) %s", x.mountpoint, err, err)
					lc.err = true
				}
			} else {
				lc.err = true
				log.Printf("Error: checker subprocess for %s failed: (%T) %s", x.mountpoint, err, err)
			}
		} else {
			lc.live = true
		}
		doneChan <- lc
	}()

	return func() *LivenessCheck {
		return <-doneChan
	}
}
