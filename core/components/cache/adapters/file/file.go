package file

type Option struct {
}

type File struct {
}

func (f *File) Get(string) (string, error) {
	panic("implement me")
}

func (f *File) SetCachePrefix(string) {
	panic("implement me")
}

func (f *File) Save(string, string) bool {
	panic("implement me")
}

func (f *File) Delete(string) bool {
	panic("implement me")
}

func (f *File) Exists(string) bool {
	panic("implement me")
}

func (f *File) SaveAll(map[string]string) bool {
	panic("implement me")
}
