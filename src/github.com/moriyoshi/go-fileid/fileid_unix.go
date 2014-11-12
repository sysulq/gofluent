// +build !windows
// +build !plan9

package fileid

import "syscall"

type FileId struct {
	Dev          uint64
	Ino          uint64
}

func GetFileId(path string, followSymlink bool) (FileId, error) {
	stat := syscall.Stat_t {}
	var err error
	if followSymlink {
		err = syscall.Stat(path, &stat)
	} else {
		err = syscall.Lstat(path, &stat)
	}
	if err != nil {
		return FileId{}, err
	}
	return FileId { uint64(stat.Dev), stat.Ino }, nil
}

func IsSame(a, b FileId) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino
}
