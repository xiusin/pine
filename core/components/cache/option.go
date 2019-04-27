package cache

type Option interface {
	Get(string) (interface{}, error)
	GetBool( string) bool
	GetDefaultBool (string, bool) bool
	GetInt(string) int
	GetDefaultInt(string, int) int
	GetString(string) string
	GetDefaultString(string, string) string
	Set(key string, val interface{}) error
}

type EmptyOption struct {

}

func (option *EmptyOption) Get(string) (interface{}, error) {
	panic("implement me")
}

func (option *EmptyOption) GetBool(string) bool {
	panic("implement me")
}

func (option *EmptyOption) GetDefaultBool(string, bool) bool {
	panic("implement me")
}

func (option *EmptyOption) GetInt(string) int {
	panic("implement me")
}

func (option *EmptyOption) GetDefaultInt(string, int) int {
	panic("implement me")
}

func (option *EmptyOption) GetString(string) string {
	panic("implement me")
}

func (option *EmptyOption) GetDefaultString(string, string) string {
	panic("implement me")
}

func (option *EmptyOption) Set(key string, val interface{}) error {
	panic("implement me")
}

func (option *EmptyOption)  {

}