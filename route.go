package router

type Route struct {
	Method     string
	Middleware []Handler
	Handle     Handler
}
