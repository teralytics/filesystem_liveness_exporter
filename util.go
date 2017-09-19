package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

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
