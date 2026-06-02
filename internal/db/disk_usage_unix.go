//go:build darwin || linux || freebsd || netbsd || openbsd

package db

import (
	"os"
	"syscall"
)

func diskUsageBytes(info os.FileInfo) int64 {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.Size()
	}
	return stat.Blocks * 512
}
