// +build ignore

package fileid

// Opaque structure whose structure may be different across OSes
type struct FileId {}

// Retrieve the FileId for the given file.
// Note that the value of followSymlink is ignored on plan9
func GetFileId(path string, followSymlink bool) (FileId, error) {}

// Compare the two FileId's and returns true if those are the same.
func IsSame(a, bFileId) bool {}
