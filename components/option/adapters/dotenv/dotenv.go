package dotenv

import "github.com/xiusin/router/components/option"

type DotEnv struct {
	option.Option
}


func New(setter ...option.OptionSetter) *Option {
	opt := Default()
	for k := range setter {
		setter[k](opt)
	}
	return opt
}

