package GlobalStore

import (
	"errors"
	"github.com/KusionStack/kpm/pkg/go-oneutils/Fetch"
	"github.com/KusionStack/kpm/pkg/go-oneutils/PathHandle"
	"os"
	"path/filepath"
	"strings"
)

const Separator = string(filepath.Separator)
const EmptyString = ""
const (
	hashStrMod = iota
	hashStrPrefix
)

type FileStore struct {
	//工作根目录
	root string
	//元数据目录（在根目录下的相对目录）
	metadata string
	//构建目录
	build string
	//存储目录
	store string
	//桶数量，建议为16的次方
	bucketCount int
	//桶数量指数，16的指数次方
	bucketCountIndexNumber int
	//桶分配方式 hashStrMod hashStrPrefix
	bucketAllocationMethod int
	//hash算法 建议sha512
	bucketHashType string
	//忽略规则方法
	ignoreFunc []IgnoreFunc
}
type FileStoreConfig struct {
	//工作根目录
	Root string
	//元数据目录（在根目录下的相对目录）
	Metadata string
	//构建目录（在根目录下的相对目录）
	Build string
	//存储目录（在根目录下的相对目录）
	Store string
	//桶数量指数，16的指数次方
	BucketCountIndexNumber int
	//桶分配方式 hashStrMod,hashStrPrefix
	BucketAllocationMethod string
	//hash算法 建议sha512
	BucketHashType string
}

// IgnoreFunc 忽略文件的规则 返回真为忽略
type IgnoreFunc func(relPath string, info os.FileInfo) bool

func IgnoreDotGitPath(relPath string, info os.FileInfo) bool {
	if strings.HasPrefix(relPath, ".git"+Separator) {
		return true
	}
	return false
}
func IgnoreDotExternalPath(relPath string, info os.FileInfo) bool {
	if strings.HasPrefix(relPath, "external"+Separator) {
		return true
	}
	return false
}

type FileInfo struct {
	//日期
	CheckedAt int64 `json:"checked_at"`
	//完整性校验
	Integrity Integrity `json:"integrity"`
	//权限
	Mode int `json:"mode"`
	//文件大小
	Size int64 `json:"size"`
}

type Metadata interface {
	GetFileInfoMap() FileInfoMap
	SetFileInfoMap(fim FileInfoMap)
	Fetch.EasyJsonSerialization
}

// NewFileStore 初始化文件存储
func NewFileStore(cfg FileStoreConfig, ignoreFunc ...IgnoreFunc) (*FileStore, error) {
	f := FileStore{
		root:       cfg.Root,
		metadata:   cfg.Root + Separator + "metadata",
		build:      cfg.Root + Separator + "build",
		store:      cfg.Root + Separator + "store",
		ignoreFunc: ignoreFunc,
	}
	if cfg.Metadata != "" {
		f.metadata = cfg.Root + Separator + cfg.Metadata
	}
	if cfg.Build != "" {
		f.build = cfg.Root + Separator + cfg.Build
	}
	if cfg.Store != "" {
		f.store = cfg.Root + Separator + cfg.Store
	}
	err := PathHandle.KeepDirsExist(
		f.root,
		f.metadata,
		f.build,
		f.store,
	)
	if err != nil {
		return nil, nil
	}
	switch cfg.BucketCountIndexNumber {
	case 3:
		f.bucketCount = 4096
		f.bucketCountIndexNumber = 3
	default:
		f.bucketCount = 256
		f.bucketCountIndexNumber = 2
	}
	switch cfg.BucketAllocationMethod {
	case "hashStrPrefix":
		f.bucketAllocationMethod = hashStrPrefix
	default:
		//hashStrMod
		f.bucketAllocationMethod = hashStrMod
	}
	switch cfg.BucketHashType {
	case "md5":
		f.bucketHashType = "md5"
	case "sha1":
		f.bucketHashType = "sha1"
	case "sha256":
		f.bucketHashType = "sha256"
	default:
		//case "sha512":
		f.bucketHashType = "sha512"
	}
	return &f, nil
}

// contains  判断path1是否包含path2
func contains(path1, path2 string) bool {
	if strings.HasPrefix(path1, path2) {
		return true
	}
	tmp, tmp2 := strings.Split(path2, Separator), ""
	for i := 0; i < len(tmp); i++ {
		tmp2 += tmp[i]
		//判断是否相等
		if tmp2 == path1 {
			return true
		}
		tmp2 += Separator
	}
	return false
}

// AddDir 添加目录到全局文件存储
func (s FileStore) AddDir(absPath string) (FileInfoMap, error) {
	//路径不应该为root目录的父目录，及不应该为相对路径
	if !filepath.IsAbs(absPath) {
		return nil, errors.New("the path is not absolute")
	}
	if contains(absPath, s.store) {
		return nil, errors.New("absPath cannot contain store root directory")
	}
	fim := AcquireFileInfoMap()
	err := filepath.Walk(absPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			//获取相对路径到结构体切片
			if info.IsDir() {
				//跳过文件夹
				return nil
			}
			rel, err := filepath.Rel(absPath, path)
			if err != nil {
				return err
			}
			for i := 0; i < len(s.ignoreFunc); i++ {
				if s.ignoreFunc[i](rel, info) {
					return nil
				}
			}
			file, err2 := os.ReadFile(path)
			if err2 != nil {
				return err2
			}

			rp := []byte(rel)
			//统一为Linux下的分隔符
			PathHandle.UnifyPathSlashSeparator(rp)
			//添加文件信息
			it := NewIntegrity(s.bucketHashType, file)
			afi := AcquireFileInfo()
			afi.CheckedAt = info.ModTime().Unix()
			afi.Integrity = it
			afi.Mode = int(info.Mode())
			afi.Size = info.Size()
			fim[string(rp)] = afi
			switch s.bucketAllocationMethod {
			case hashStrPrefix:

				hashString, err := it.GetRawHashString()
				if err != nil {
					return err
				}
				//println(hashString, hashString[:2], hashString[2:])
				bucketpath := s.store + Separator + hashString[:s.bucketCountIndexNumber]
				err = PathHandle.KeepDirExist(bucketpath)
				if err != nil {
					return err
				}
				err = os.WriteFile(bucketpath+Separator+hashString[s.bucketCountIndexNumber:], file, 0777)
				if err != nil {
					return err
				}

			default:
				//case :"hashStrMod"

				//添加文件到全局存储桶
				rawhashstring, mod, err := it.GetRawHashStringAndModFast(uint64(s.bucketCount))
				if err != nil {
					return err
				}
				bucketpath := s.store + Separator + mod
				err = PathHandle.KeepDirExist(bucketpath)
				if err != nil {
					return err
				}
				err = os.WriteFile(bucketpath+Separator+rawhashstring, file, 0777)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return fim, nil
}

// NewFileInfoMapFromDir 仅从文件夹生成FileInfoMap
func (s FileStore) NewFileInfoMapFromDir(absPath string) (FileInfoMap, error) {
	//路径不应该为root目录
	if !filepath.IsAbs(absPath) {
		return nil, errors.New("the path is not absolute")
	}
	if contains(absPath, s.store) {
		return nil, errors.New("absPath cannot contain store root directory")
	}
	fim := AcquireFileInfoMap()
	err := filepath.Walk(absPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			//获取相对路径到结构体切片
			if info.IsDir() {
				//跳过文件夹
				return nil
			}
			rel, err := filepath.Rel(absPath, path)
			if err != nil {
				return err
			}
			for i := 0; i < len(s.ignoreFunc); i++ {
				if s.ignoreFunc[i](rel, info) {
					return nil
				}
			}
			file, err2 := os.ReadFile(path)
			if err2 != nil {
				return err2
			}

			rp := []byte(rel)
			//统一为Linux下的分隔符
			PathHandle.UnifyPathSlashSeparator(rp)
			//添加文件信息
			afi := AcquireFileInfo()
			afi.CheckedAt = info.ModTime().Unix()
			afi.Integrity = NewIntegrity(s.bucketHashType, file)
			afi.Mode = int(info.Mode())
			afi.Size = info.Size()
			fim[string(rp)] = afi
			return nil
		})
	if err != nil {
		return nil, err
	}
	return fim, nil
}

// BuildDir 通过FileInfoMap构建包目录在工作区构建目录下
func (s FileStore) BuildDir(fim FileInfoMap, pkgDir string) error {
	//构建目录
	buildpath := s.build + Separator + PathHandle.URLToLocalDirPath(pkgDir)
	exists, err := PathHandle.DirExist(buildpath)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("path exist")
	}
	err = os.MkdirAll(buildpath, os.ModePerm)
	if err != nil {
		return err
	}
	for s2, info := range fim {
		d, _ := filepath.Split(s2)
		if d != "" {
			err = PathHandle.KeepDirExist(buildpath + Separator + d)
			if err != nil {
				err = os.RemoveAll(buildpath)
				if err != nil {
					return err
				}
				return err
			}
		}
		switch s.bucketAllocationMethod {
		case hashStrPrefix:
			hashString, err := info.Integrity.GetRawHashString()
			if err != nil {
				err = os.RemoveAll(buildpath)
				if err != nil {
					return err
				}
				return err
			}
			hashpath := s.store + Separator + hashString[:s.bucketCountIndexNumber] + Separator + hashString[s.bucketCountIndexNumber:]
			err = os.Link(hashpath, buildpath+Separator+s2)
			if err != nil {
				err = os.RemoveAll(buildpath)
				if err != nil {
					return err
				}
				return err
			}
		default:
			//case :"hashStrMod"
			hashString, mod, err := info.Integrity.GetRawHashStringAndModFast(uint64(s.bucketCount))
			if err != nil {
				err = os.RemoveAll(buildpath)
				if err != nil {
					return err
				}
				return err
			}
			hashpath := s.store + Separator + mod + Separator + hashString
			err = os.Link(hashpath, buildpath+Separator+s2)
			if err != nil {
				err = os.RemoveAll(buildpath)
				if err != nil {
					return err
				}
				return err
			}
		}
	}
	return nil
}

// Link 链接包目录到目标目录
func (s FileStore) Link(pkgDir, targetAbsPath string) error {
	if !filepath.IsAbs(targetAbsPath) {
		return errors.New("the path is not absolute")
	}
	var tmp string
	for i := len(targetAbsPath) - 1; i >= 0; i-- {
		if targetAbsPath[i] == filepath.Separator {
			tmp = targetAbsPath[:i]
			break
		}
	}
	err := PathHandle.KeepDirExist(tmp)
	if err != nil {
		return err
	}
	err = os.Symlink(s.build+Separator+PathHandle.URLToLocalDirPath(pkgDir), targetAbsPath)
	if err != nil {
		return err
	}
	return nil
}

// VerifyDir 验证文件目录
func (s FileStore) VerifyDir(fim FileInfoMap, absPath string) (bool, error) {
	if !filepath.IsAbs(absPath) {
		return false, errors.New("the path is not absolute")
	}
	nfim, err := s.NewFileInfoMapFromDir(absPath)
	if err != nil {
		return false, err
	}
	result := fim.FileEqual(nfim)
	ReleaseFileInfoMap(nfim)
	return result, nil
}

// GetMetadataPath 获取metadata的理论绝对路径
func (s FileStore) GetMetadataPath(pkgDir string) (string, error) {
	p := s.metadata + Separator + PathHandle.URLToLocalDirPath(pkgDir)
	dir, _ := filepath.Split(p)
	err := PathHandle.KeepDirExist(dir)
	if err != nil {
		return "", err
	}
	return p, nil
}

// GetDirPath 获取pkgDir的理论绝对路径
func (s FileStore) GetDirPath(pkgDir string) (string, error) {
	p := s.build + Separator + PathHandle.URLToLocalDirPath(pkgDir)
	dir, _ := filepath.Split(p)
	err := PathHandle.KeepDirExist(dir)
	if err != nil {
		return "", err
	}
	return p, nil
}

// DirIsExist 检测pkgDir存在
func (s FileStore) DirIsExist(pkgDir string) (bool, error) {
	return PathHandle.DirExist(s.build + Separator + PathHandle.URLToLocalDirPath(pkgDir))
}
