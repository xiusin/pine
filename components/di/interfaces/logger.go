package interfaces

/**
为了更好的实现日志， 只需要实现两个方法即可。
*/
type LoggerInf interface {
	Error(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
	Print(msg string, args ...interface{})
	Printf(format string, args ...interface{})
}
