package file_logger

import (
	"os"
	"syscall"
	"time"
)

//------------------------------------------------------------------------------

func getFileCreationtime(fi os.FileInfo) time.Time {
	stat := fi.Sys().(*syscall.Win32FileAttributeData)
	return time.Unix(0, stat.CreationTime.Nanoseconds())
}
