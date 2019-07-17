package router

import "strconv"

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
func (c *Params) GetDefault(key, defaultVal string) string {
	val := c.Get(key)
	if val != "" {
		return val
	}
	return defaultVal
}

func (c *Params) GetInt(key string, defaultVal ...int) (val int, res bool) {
	val, err := strconv.Atoi(c.Get(key))
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Params) GetInt64(key string, defaultVal ...int64) (val int64, res bool) {
	val, err := strconv.ParseInt(c.Get(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Params) GetFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	val, err := strconv.ParseFloat(c.Get(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}
