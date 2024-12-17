package contracts

type Logger interface {
	Debug(string, ...any)
	Warn(string, ...any)
	Info(string, ...any)
	Error(string, ...any)
}
