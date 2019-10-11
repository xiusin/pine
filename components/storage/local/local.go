package local

import (
	"github.com/xiusin/router/components/storage"
	"github.com/xiusin/router/utils"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type Option struct {
	Domain              string // 本地目录,自动追加最后面的/字符
	LocalBaseDir        string // 本地目录,自动追加最后面的/字符
	PutReturnWithDomain bool
}

func (o *Option) GetEndpoint() string {
	return o.Domain
}

type Local struct {
	option *Option
}

func (o *Local) PutFromReader(storeFilePath string, localPathReader io.Reader) (string, error) {
	f, err := os.OpenFile(o.getAbsPath(storeFilePath), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, localPathReader)
	if err != nil {
		return "", err
	}
	if o.option.PutReturnWithDomain {
		return o.option.Domain + storeFilePath, nil
	} else {
		return storeFilePath, nil
	}
}

func (o *Local) PutFromFile(storeFilePath, filePath string) (string, error) {
	return "", nil
}

func (o *Local) Delete(storeFilePath string) error {
	return os.Remove(o.getAbsPath(storeFilePath))
}

func (o *Local) getAbsPath(path string) string {
	return o.option.LocalBaseDir + path
}

func (o *Local) Exists(storageFilePath string) (bool, error) {
	if utils.FileExists(o.getAbsPath(storageFilePath)) {
		return true, nil
	}
	return false, nil
}

func (o *Local) List(dir ...string) (fs []os.FileInfo, err error) {
	if len(dir) == 0 {
		dir = append(dir, "")
	}
	fullDir := o.option.LocalBaseDir + dir[0]
	fs, err = ioutil.ReadDir(fullDir)
	return
}

func init() {
	storage.Register("local", func(option storage.Option) storage.IStorage {
		opt := option.(*Option)
		opt.LocalBaseDir = strings.TrimRight(opt.LocalBaseDir, "/") + "/"
		opt.Domain = strings.TrimRight(opt.Domain, "/") + "/"
		if !utils.IsDir(opt.LocalBaseDir) {
			if err := os.MkdirAll(opt.LocalBaseDir, 0644); err != nil {
				panic(err)
			}
		}
		return &Local{option: opt}
	})
}
