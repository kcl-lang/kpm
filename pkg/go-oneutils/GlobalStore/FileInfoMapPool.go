package GlobalStore

import "sync"

var poolFileInfoMap = sync.Pool{
	New: func() any {
		return make(FileInfoMap, 16)
	},
}

func (m FileInfoMap) Reset() {
	for k, v := range m {
		ReleaseFileInfo(v)
		delete(m, k)
	}
}
func AcquireFileInfoMap() FileInfoMap {
	return poolFileInfoMap.Get().(FileInfoMap)
}

func ReleaseFileInfoMap(m FileInfoMap) {
	m.Reset()
	poolFileInfoMap.Put(m)
}
