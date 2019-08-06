package interfaces

/**
为了更好的实现日志
*/
type ILogger interface {
	Error(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
	Print(msg string, args ...interface{})
	Printf(format string, args ...interface{})
}
