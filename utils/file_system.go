package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

type File struct {
	path string
}

func NewFile(path string) (*File, error) {
	if FileExists(path) {
		return &File{path: path}, nil
	}
	return nil, errors.New(fmt.Sprintf("%s is not exists", path))
}

func (f *File) Ext() string {
	return path.Ext(f.path)
}

func (f *File) Name() (string, error) {
	fi, err := os.Stat(f.path)
	if err != nil {
		return "", err
	}
	return fi.Name(), nil
}

func (f *File) FileSize() (int64, error) {
	fi, err := os.Stat(f.path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func (f *File) Dir() string {
	return path.Dir(f.path)
}

func (f *File) CopyTo(name string) error {
	return Copy(f.path, name)
}

func (f *File) Rename(name string) error {
	if err := Rename(f.path, name); err != nil {
		return err
	}
	f.path = name //重置地址到新的位置
	return nil
}

func IsDir(dirname string) bool {
	f, err := os.Stat(dirname)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return f.IsDir()
}

func FileExists(path string) bool {
	f, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return !f.IsDir()
}

func Rename(src, target string) error {
	return os.Rename(src, target)
}

func Copy(src, target string) error {
	srcSource, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcSource.Close()
	targetSource, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer targetSource.Close()
	_, err = io.Copy(srcSource, targetSource)
	return err
}
