package go_logger

import (
	"os"
	"syscall"
	"time"
)

//------------------------------------------------------------------------------

func getFileCreationTime(fi os.FileInfo) time.Time {
	stat := fi.Sys().(*syscall.Dir)
	return time.Unix(int64(stat.Mtime), 0)
}
