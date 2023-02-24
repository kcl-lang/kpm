package GlobalStore

import "sync"

var poolFileInfo = sync.Pool{
	New: func() any {
		return &FileInfo{}
	},
}

func (fi *FileInfo) Reset() {
	fi.CheckedAt = 0
	fi.Mode = 0
	fi.Size = 0
	fi.Integrity = EmptyString
}
func AcquireFileInfo() *FileInfo {
	return poolFileInfo.Get().(*FileInfo)
}

func ReleaseFileInfo(fi *FileInfo) {
	fi.Reset()
	poolFileInfo.Put(fi)
}
