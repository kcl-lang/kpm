package kpm

import (
	"bufio"
	"bytes"
	"github.com/KusionStack/kpm/pkg/go-oneutils/Convert"
	"github.com/KusionStack/kpm/pkg/go-oneutils/Set"
	"github.com/valyala/bytebufferpool"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type KCLFile struct {
	//文件路径
	Path string
	//文件内容
	Body []byte
	//导入集合
	Imports Set.Set
	//写入标志
	WriterFlag bool
}

func NewKCLFileFromFile(path string) (*KCLFile, error) {
	kclF := &KCLFile{
		Path:       path,
		Body:       nil,
		Imports:    Set.New(),
		WriterFlag: false,
	}
	//读取每一行
	rawbody, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	//rawbody := Convert.S2B(Def)
	r := bufio.NewReader(bytes.NewReader(rawbody))
	linecount := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			//遇到未知错误
			panic(err)
		}
		if err == io.EOF {
			break
		}
		linecount++
		line = strings.TrimSpace(line)
		linelen := len(line)

		if linelen <= 6 {
			//跳过无效短行
			continue
		}
		if line[0:1] == "#" {
			//跳过注释
			continue
		}
		if line[0:6] != "import" {
			//遇到非import直接退出
			break
		}
		lastbyte := ' '
		var modname []byte
		loadmodname := false
		for i := 6; i < linelen; i++ {
			//当上一个元素是空格，这个元素不是空格的时候跳入

			if !loadmodname {
				if lastbyte == ' ' && line[i] != ' ' {
					loadmodname = true
					lastbyte = int32(line[i])
					modname = append(modname, line[i])
					continue
				}
			}
			//当上一个元素不是空格，这个元素是空格的时候跳出
			if lastbyte != ' ' && line[i] == ' ' {
				break
			}
			modname = append(modname, line[i])
			lastbyte = int32(line[i])
		}
		//检查是否是系统模块
		issysmod := false
		modnamestr := string(modname[1:])
		for i := 0; i < len(systemPkg); i++ {
			if systemPkg[i] == modnamestr {
				issysmod = true
				break

			}
		}
		if issysmod {
			kclF.Imports.SAdd(modnamestr)
		}

	}
	for i := 0; i >= len(rawbody) || linecount > 0; i++ {
		if linecount == 1 {
			kclF.Body = rawbody[i:]
			break
		}
		if rawbody[i] == '\n' {
			linecount--
		}

	}
	return kclF, nil
}
func (f *KCLFile) Save() error {
	if f.WriterFlag {
		ipt := f.Imports.SMembers()
		b := bytebufferpool.Get()
		defer bytebufferpool.Put(b)
		for i := 0; i < len(ipt); i++ {
			b.WriteString("import ")
			b.WriteString(ipt[i])
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
		b.Write(f.Body)
		err := os.WriteFile(f.Path, b.Bytes(), 0777)
		if err != nil {
			return err
		}
	}
	return nil
}
func (f *KCLFile) AddPkgPrefix(str string) {
	ipt := f.Imports.SMembers()
	for i := 0; i < len(ipt); i++ {
		f.WriterFlag = true
		ipt[i] = str + "." + ipt[i]
	}
	if f.WriterFlag {
		f.Imports.Reset()
		f.Imports.SAdd(ipt...)
	}
}
func (f *KCLFile) AddPkgPrefixFromPath(str string) {
	var newPrefix []byte
	for i := 0; i < len(str); i++ {
		tmp := str[i]
		if tmp != filepath.Separator {
			newPrefix = append(newPrefix, tmp)
		} else {
			newPrefix = append(newPrefix, '.')
		}

	}
	ipt := f.Imports.SMembers()
	for i := 0; i < len(ipt); i++ {
		f.WriterFlag = true
		ipt[i] = Convert.B2S(newPrefix) + "." + ipt[i]
	}
	if f.WriterFlag {
		f.Imports.Reset()
		f.Imports.SAdd(ipt...)
	}
}
