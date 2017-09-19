package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// executable returns the path to the current executable.
func executable() (string, error) {
	switch runtime.GOOS {
	case "linux", "android":
		const deletedTag = " (deleted)"
		execpath, err := os.Readlink("/proc/self/exe")
		if err != nil {
			return execpath, err
		}
		execpath = strings.TrimSuffix(execpath, deletedTag)
		execpath = strings.TrimPrefix(execpath, deletedTag)
		return execpath, nil
	case "netbsd":
		return os.Readlink("/proc/curproc/exe")
	case "dragonfly":
		return os.Readlink("/proc/curproc/file")
	case "solaris":
		return os.Readlink(fmt.Sprintf("/proc/%d/path/a.out", os.Getpid()))
	}
	return "", errors.New("ExecPath not implemented for " + runtime.GOOS)
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

// Filesystem represents a file system as it is mounted on the
// system at the time DiscoverFilesystems is invoked.
type Filesystem struct {
	device     string
	mountpoint string
	fstype     string
}

// DiscoverFilesystems discovers file systems as specified in the
// mounts file (usually /proc/mounts).
//
// It returns a list of *Filesystem that specifies the device,
// the mount point, and the file system type, of all mounted
// devices, so long as they match the allowed file system
// types passed to this function (or all file systems, if the
// allowedFsTypes list is empty).
func DiscoverFilesystems(mountsFile string, allowedFsTypes []string) []*Filesystem {
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
