package PathHandle

import (
	"crypto/sha512"
	"errors"
	"github.com/KusionStack/kpm/pkg/go-oneutils/Convert"
	"github.com/KusionStack/kpm/pkg/go-oneutils/Random"
	"github.com/cespare/xxhash/v2"
	"os"
	"path/filepath"
	"strings"
)

const Separator = string(filepath.Separator)
const hextable = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RunInTempDir 在缓存目录中工作
func RunInTempDir(f func(tmppath string) error) error {
	p := os.TempDir() + Separator + Convert.B2S(Random.RandBytes32())
	err := KeepDirExist(p)
	if err != nil {
		return err
	}
	defer os.RemoveAll(p)

	err = f(p)
	if err != nil {
		return err
	}

	return nil
}

// UnifyPathSlashSeparator 统一路径分隔符为斜杠
func UnifyPathSlashSeparator(b []byte) {
	for i := 0; i < len(b); i++ {
		if b[i] == '\\' {
			b[i] = '/'
		}
	}
}

// UnifyPathBackSlashSeparator 统一路径分隔符为反斜杠
func UnifyPathBackSlashSeparator(b []byte) {
	for i := 0; i < len(b); i++ {
		if b[i] == '/' {
			b[i] = '\\'
		}
	}
}

// UnifyPathBackLocalSeparator 统一路径分隔符为当前系统斜杠
func UnifyPathBackLocalSeparator(b []byte) {
	for i := 0; i < len(b); i++ {
		if b[i] == '/' || b[i] == '\\' {
			b[i] = filepath.Separator
		}
	}
}

// Bucket256AllocatedUseSha512  使用sha512作为hash算法为文件分配桶（256个桶下）
//
//	mod取自hash转换为可见字符串之后的值
//	 已优化 可以被内联
func Bucket256AllocatedUseSha512(fileBytes []byte) (hash string, mod string) {
	sha512hash, dst, j := sha512.Sum512(fileBytes), make([]byte, 128), 0
	//转为可见字符
	for _, v := range sha512hash {
		dst[j], dst[j+1] = hextable[v>>4], hextable[v&0x0f]
		j += 2
	}
	//
	t, hash := xxhash.Sum64(dst)%256, string(dst)
	mod = string([]byte{hextable[t>>4], hextable[t%16]})
	return
}

// KeepDirsExist  确保某些目录一定存在
func KeepDirsExist(paths ...string) error {
	for i := 0; i < len(paths); i++ {
		err := KeepDirExist(paths[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// KeepDirExist 确保某目录一定存在
func KeepDirExist(path string) error {
	f, err := os.Stat(path)
	if err == nil {
		//存在
		if f.IsDir() {
			//是已经存在的目录 不处理
			return nil
		}
		//是文件,返回错误
		return errors.New("file exists instead of directory")
	}
	if os.IsNotExist(err) {

		//不存在
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
		return nil
	}
	//其他错误
	return err
}

// URLToLocalDirPath uri转本地相对路径
func URLToLocalDirPath(url string) string {
	tmp := strings.Split(url, "://")
	tmplen := len(tmp)
	var tmpbytes []byte
	if tmplen == 2 {
		tmpbytes = append(tmpbytes, tmp[1]...)

	} else {
		tmpbytes = append(tmpbytes, tmp[0]...)
	}
	for i := 0; i < len(tmpbytes); i++ {
		if tmpbytes[i] == '/' {
			tmpbytes[i] = filepath.Separator
		}
	}
	return string(tmpbytes)
}

// URLToLocalDirPathNoHost  uri转本地相对路径不带域名
func URLToLocalDirPathNoHost(url string) string {
	tmp := strings.Split(url, "://")
	tmplen := len(tmp)
	var tmpbytes []byte
	if tmplen == 2 {
		tmpbytes = append(tmpbytes, tmp[1]...)

	} else {
		tmpbytes = append(tmpbytes, tmp[0]...)
	}
	hostflag := 0
	hostindex := 0
	for i := 0; i < len(tmpbytes); i++ {
		if tmpbytes[i] == '/' {
			hostflag++
			if hostflag == 1 {
				hostindex = i
			}
			tmpbytes[i] = filepath.Separator

		}
	}
	if hostflag == 0 {
		return ""
	}
	return string(tmpbytes[hostindex:])
}

// PathExist 路径是否存在
func PathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// FileExist  文件是否存在
func FileExist(path string) (bool, error) {
	f, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if f.IsDir() {
		return false, errors.New("it is a Dir")
	}
	return true, nil
}

// DirExist 目录是否存在
func DirExist(path string) (bool, error) {
	f, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !f.IsDir() {
		return false, errors.New("it is a File")
	}
	return true, nil
}
