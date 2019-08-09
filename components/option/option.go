package option

import (
	"github.com/gorilla/securecookie"
	"time"

	"github.com/spf13/viper"
)

const (
	DevMode = iota
	ProdMode
)

type (
	cookieOption struct {
		Secure     bool
		HttpOnly   bool
		Path       string
		HashKey    string
		BlockKey   string
		Serializer securecookie.Serializer
	}

	Option struct {
		maxMultipartMemory int64
		reqTimeOutMessage  string
		csrfLifeTime       time.Duration
		serverName         string
		csrfName           string
		certfile           string
		keyfile            string
		timeout            time.Duration
		cookie             cookieOption
		port               int
		host               string
		gzip               bool
		env                int
	}
)

type optionSetter func(o *Option)

func New(setter ...optionSetter) *Option {
	opt := Default()
	for k := range setter {
		setter[k](opt)
	}
	return opt
}

func Default() *Option {
	opt := &Option{
		gzip:               false,
		port:               9528,
		host:               "0.0.0.0",
		reqTimeOutMessage:  "request timeout",
		timeout:            time.Second * 3,
		env:                DevMode,
		serverName:         "xiusin/router",
		csrfName:           "csrf_token",
		csrfLifeTime:       time.Second * 30,
		maxMultipartMemory: 8 << 20,
		cookie: cookieOption{
			Secure:     false,
			HttpOnly:   false,
			Path:       "/",
			HashKey:    "ROUTER-HASH-KEY",
			BlockKey:   "ROUTER-BLOCK-KEY",
			Serializer: &securecookie.GobEncoder{},
		},
	}
	return opt
}

// 参数注入到viper内
func (o *Option) ToViper() {
	if o.IsDevMode() {
		viper.Debug()
	}
	AddGlobal("csrf_name", o.csrfName)
	AddGlobal("csrf_lifetime", o.csrfLifeTime)
	AddGlobal("cookie.secure", o.cookie.Secure)
	AddGlobal("cookie.http_only", o.cookie.HttpOnly)
	AddGlobal("cookie.path", o.cookie.Path)
	AddGlobal("cookie.hash_key", o.cookie.HashKey)
	AddGlobal("cookie.block_key", o.cookie.BlockKey)
	AddGlobal("cookie.serializer", o.cookie.Serializer)
	AddGlobal("env", o.env)
}

func (o *Option) GetEnv() int {
	return o.env
}

func (o *Option) GetPort() int {
	return o.port
}

func (o *Option) GetMaxMultipartMemory() int64 {
	return o.maxMultipartMemory
}

func (o *Option) GetGzip() bool {
	return o.gzip
}

func (o *Option) GetReqTimeOutMessage() string {
	return o.reqTimeOutMessage
}

func (o *Option) GetHost() string {
	return o.host
}

func (o *Option) GetServerName() string {
	return o.serverName
}

func (o *Option) GetCsrfName() string {
	return o.csrfName
}

func (o *Option) GetCsrfLiftTime() time.Duration {
	return o.csrfLifeTime
}

func (o *Option) GetTimeOut() time.Duration {
	return o.timeout
}

func (o *Option) GetCertFile() string {
	return o.certfile
}

func (o *Option) GetKeyFile() string {
	return o.keyfile
}

func (o *Option) IsDevMode() bool {
	return o.env == DevMode
}

func (o *Option) IsProdMode() bool {
	return o.env == ProdMode
}

func AddGlobal(key string, val interface{}) {
	viper.Set(key, val)
}

func OptEnvMode(env int) func(o *Option) {
	return func(o *Option) {
		o.env = env
	}
}
func OptCsrfName(name string) func(o *Option) {
	return func(o *Option) {
		o.csrfName = name
	}
}

func OptCsrfLifeTime(lifttime time.Duration) func(o *Option) {
	return func(o *Option) {
		o.csrfLifeTime = lifttime
	}
}

func OptPort(port int) func(o *Option) {
	return func(o *Option) {
		o.port = port
	}
}

func OptTimeOut(dur time.Duration) func(o *Option) {
	return func(o *Option) {
		o.timeout = dur
	}
}

func OptServerName(sername string) func(o *Option) {
	return func(o *Option) {
		o.serverName = sername
	}
}

func OptReqTimeOutMessage(message string) func(o *Option) {
	return func(o *Option) {
		o.reqTimeOutMessage = message
	}
}

func OptMaxMultipartMemory(mem int64) func(o *Option) {
	return func(o *Option) {
		o.maxMultipartMemory = mem
	}
}
func OptCookieSecure(secure bool) func(o *Option) {
	return func(o *Option) {
		o.cookie.Secure = secure
	}
}

func OptCookieHttpOnly(http bool) func(o *Option) {
	return func(o *Option) {
		o.cookie.Secure = http
	}
}

func OptCookieHashKey(hash string) func(o *Option) {
	return func(o *Option) {
		o.cookie.HashKey = hash
	}
}

func OptCookieBlockKey(block string) func(o *Option) {
	return func(o *Option) {
		o.cookie.HashKey = block
	}
}

func OptCertFile(file string) func(o *Option) {
	return func(o *Option) {
		o.certfile = file
	}
}

func OptKeyFile(file string) func(o *Option) {
	return func(o *Option) {
		o.keyfile = file
	}
}

func OptGzip(gzip bool) func(o *Option) {
	return func(o *Option) {
		o.gzip = gzip
	}
}
