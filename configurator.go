package pine

type Configuration struct {
	maxMultipartMemory        int64
	serverName                string
	charset                   string
	withoutFrameworkLog       bool
	autoParseControllerResult bool
	autoParseForm             bool
}

var configuration = Configuration{}

type Configurator func(o *Configuration)

func WithServerName(srvName string) Configurator {
	return func(o *Configuration) {
		o.serverName = srvName
	}
}

func WithAutoParseForm(autoParseForm bool) Configurator {
	return func(o *Configuration) {
		o.autoParseForm = autoParseForm
	}
}

func WithMaxMultipartMemory(mem int64) Configurator {
	return func(o *Configuration) {
		o.maxMultipartMemory = mem
	}
}

func WithoutFrameworkLog(hide bool) Configurator {
	return func(o *Configuration) {
		o.withoutFrameworkLog = hide
	}
}

func WithCharset(charset string) Configurator {
	return func(o *Configuration) {
		o.charset = charset
	}
}

func WithAutoParseControllerResult(auto bool) Configurator {
	return func(o *Configuration) {
		o.autoParseControllerResult = auto
	}
}
