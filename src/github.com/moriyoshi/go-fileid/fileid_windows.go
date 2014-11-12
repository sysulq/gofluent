// +build windows
package fileid

import "syscall"

type FileId struct {
	VolumeSerial uint32
	Index        uint64
}

const FILE_FLAG_OPEN_REPARSE_POINT = uint32(0x00200000)

func GetFileId(path string, followSymlink bool) (FileId, error) {
	upath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return FileId {}, err
	}
	flag := uint32(0)
	if !followSymlink {
		flag |= FILE_FLAG_OPEN_REPARSE_POINT
	}
	hFile, err := syscall.CreateFile(
		upath, 0, 0, nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS | flag, 0,
	)
	if err != nil {
		return FileId {}, err
	}
	info := syscall.ByHandleFileInformation {}
	err = syscall.GetFileInformationByHandle(syscall.Handle(hFile), &info)
	syscall.CloseHandle(hFile)
	if err != nil {
		return FileId {}, err
	}
	return FileId {
		VolumeSerial: info.VolumeSerialNumber,
		Index: (uint64(info.FileIndexHigh) << 32) | uint64(info.FileIndexLow),
	}, nil
}

func IsSame(a, b FileId) bool {
	return a.VolumeSerial == b.VolumeSerial && a.Index == b.Index
}
