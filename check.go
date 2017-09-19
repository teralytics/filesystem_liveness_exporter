package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Filesystem struct {
	device     string
	mountpoint string
	fstype     string
}

type LivenessCheck struct {
	err      bool
	live     bool
	duration float64
}

type CheckResult struct {
	filesystem *Filesystem
	check      *LivenessCheck
}

func (x *Filesystem) Check(timeout time.Duration, optReadFile string) func() *LivenessCheck {
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

// discoverFilesystems discovers file systems as specified in the
// mounts file (usually /proc/mounts).
//
// It returns a list of *Filesystem that specifies the device,
// the mount point, and the file system type, of all mounted
// devices, so long as they match the allowed file system
// types passed to this function (or all file systems, if the
// allowedFsTypes list is empty).
func discoverFilesystems(mountsFile string, allowedFsTypes []string) []*Filesystem {
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

func CollectMetrics(timeout time.Duration, fsTypes []string, optReadFile string) []*CheckResult {
	fses := discoverFilesystems("/proc/mounts", fsTypes)
	waits := make(map[*Filesystem]func() *LivenessCheck)
	for _, fs := range fses {
		waits[fs] = fs.Check(timeout, optReadFile)
	}
	res := []*CheckResult{}
	for _, fs := range fses {
		res = append(res, &CheckResult{fs, waits[fs]()})
	}
	return res
}
