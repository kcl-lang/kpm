package GlobalStore

import (
	"github.com/KusionStack/kpm/pkg/go-oneutils/Convert"
	"sort"
)

// FileInfoMap 文件相对路径 文件信息
type FileInfoMap map[string]*FileInfo

// Equal 检测两个FileInfoMap是否相等
func (m FileInfoMap) Equal(fim FileInfoMap) bool {
	if len(m) != len(fim) {
		return false
	}
	for k, info := range m {
		fileInfo, ok := fim[k]
		if !ok {
			return false
		}
		if info.Integrity != fileInfo.Integrity {
			return false
		}
		if info.Size != fileInfo.Size {
			return false
		}
		if info.Mode != fileInfo.Mode {
			return false
		}
		if info.CheckedAt != fileInfo.CheckedAt {
			return false
		}
	}
	return true
}

// FileEqual 仅检测两个FileInfoMap的文件是否相等
func (m FileInfoMap) FileEqual(fim FileInfoMap) bool {
	if len(m) != len(fim) {
		return false
	}
	for k, info := range m {
		fileInfo, ok := fim[k]
		if !ok {
			return false
		}
		if info.Integrity != fileInfo.Integrity {
			return false
		}
		if info.Size != fileInfo.Size {
			return false
		}

	}
	return true
}

// GetIntegrity 生成这个FileInfoMap唯一的Integrity
func (m FileInfoMap) GetIntegrity() Integrity {
	t := "sha512"
	var sumlists = make([]string, len(m))
	for s, info := range m {
		value, err := info.Integrity.GetRawHashValue()
		if err != nil {
			return ""
		}
		t = info.Integrity.GetType()
		nameit := NewIntegrity(t, []byte(s))
		hashValue, err := nameit.GetRawHashValue()
		if err != nil {
			return ""
		}
		hashValue = append(hashValue, value...)
		sumlists = append(sumlists, Convert.B2S(hashValue))
	}
	sort.Strings(sumlists)
	var sumstr string
	for i := 0; i < len(sumlists); i++ {
		sumstr += sumlists[i]
	}
	return NewIntegrity(t, Convert.S2B(sumstr))
}
