//go:build windows

package db

import "os"

func diskUsageBytes(info os.FileInfo) int64 {
	return info.Size()
}
