package http

type Params struct {
	data map[string]string
}

func NewParams(data map[string]string) *Params {
	return &Params{data}
}

// 设置路由参数
func (c *Params) Set(key, value string) {
	c.data[key] = value
}

// 获取路由参数
func (c *Params) Get(key string) string {
	value, _ := c.data[key]
	return value
}

// 获取路由参数,如果为空字符串则返回 defaultVal
func (c *Params) GetParamDefault(key, defaultVal string) string {
	val := c.Get(key)
	if val != "" {
		return val
	}
	return defaultVal
}
