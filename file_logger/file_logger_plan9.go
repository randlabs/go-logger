package file_logger

import (
	"os"
	"time"
)

//------------------------------------------------------------------------------

func getFileCreationtime(fi os.FileInfo) time.Time {
	stat := fi.Sys().(*syscall.Dir)
	return time.Unix(int64(stat.Mtime), 0)
}
