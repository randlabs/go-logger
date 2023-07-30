package go_logger

import (
	"os"
	"syscall"
	"time"
)

//------------------------------------------------------------------------------

func getFileCreationTime(fi os.FileInfo) time.Time {
	stat := fi.Sys().(*syscall.Stat_t)
	return time.Unix(stat.Ctime, stat.CtimeNsec)
}
