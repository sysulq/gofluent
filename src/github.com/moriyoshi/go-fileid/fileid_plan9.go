// +build plan9

package fileid

import "syscall"

type FileId struct {
	Path       uint64
	Vers       uint32
	FileType   uint8
	ServerType uint16
	Dev        uint32
}

func GetFileId(path string, followSymlink bool) (FileId, error) {
	statBuf := make([]byte, syscall.STATFIXLEN + (16 * 4 + 127) & ^128)
	n, err := syscall.Stat(path, statBuf)
	if err != nil {
		return FileId{}, err
	}
	if n < syscall.STATFIXLEN {
		return FileId{}, syscall.ErrShortStat
	}
	realSize := int(uint16(statBuf[0]) | (uint16(statBuf[1]) << 8))
	if n < realSize {
		statBuf = make([]byte, realSize)
		n, err = syscall.Stat(path, statBuf)
		if err != nil {
			return FileId{}, err
		}
		if n < syscall.STATFIXLEN {
			panic("WTF?")
		}
	}
	dirInfo, err := syscall.UnmarshalDir(statBuf[:n])
	if err != nil {
		return FileId{}, err
	}
	return FileId { dirInfo.Qid.Path, dirInfo.Qid.Vers, dirInfo.Qid.Type, dirInfo.Type, dirInfo.Dev }, nil
}

func IsSame(a, b FileId) bool {
	return a.ServerType == b.ServerType && a.Dev == b.Dev && a.Path == b.Path
}
