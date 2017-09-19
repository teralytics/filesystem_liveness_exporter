package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
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

	start := time.Now()
	lc := &LivenessCheck{}
	myself, err := executable()
	if err != nil {
		log.Printf("Error: cannot find myself: (%T) %s", err, err)
		lc.skip = true
		lc.duration = float64(time.Now().Sub(start)) / 1000000000
		return func() *LivenessCheck { return lc }
	}

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	cmd := exec.CommandContext(ctx, myself, x.mountpoint)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		verboseLog("Starting liveness check of %s", x.mountpoint)
		err = cmd.Run()
		verboseLog("Ended liveness check of %s", x.mountpoint)
		lc.duration = float64(time.Now().Sub(start)) / 1000000000
		if err != nil {
			log.Printf("Error: checker subprocess failed to read %s: (%T) %s", x.mountpoint, err, err)
			if msg, ok := err.(*exec.ExitError); ok {
				if msg.Sys().(syscall.WaitStatus).ExitStatus() == 4 {
					lc.skip = true
				}
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

func CollectMetrics(timeout time.Duration, fsTypes []string) []*CheckResult {
	fses := discoverFilesystems("/proc/mounts", fsTypes)
	waits := make(map[*Filesystem]func() *LivenessCheck)
	for _, fs := range fses {
		waits[fs] = fs.Check(timeout)
	}
	res := []*CheckResult{}
	for _, fs := range fses {
		res = append(res, &CheckResult{fs, waits[fs]()})
	}
	return res
}
