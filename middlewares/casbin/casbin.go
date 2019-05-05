package casbin

import (
	"github.com/xiusin/router/core"
	"net/http"

	"github.com/casbin/casbin"
)

// NewAuthorizer returns the authorizer, uses a Casbin enforcer as input
func New(e *casbin.Enforcer) core.Handler {
	a := &BasicAuthorizer{enforcer: e}
	return func(c *core.Context) {
		if !a.CheckPermission(c.Request()) {
			a.RequirePermission(c)
		}
	}
}

// BasicAuthorizer stores the casbin handler
type BasicAuthorizer struct {
	enforcer *casbin.Enforcer
}

// GetUserName gets the user name from the request.
// Currently, only HTTP basic authentication is supported
func (a *BasicAuthorizer) GetUserName(r *http.Request) string {
	username, _, _ := r.BasicAuth()
	return username
}

// CheckPermission checks the user/method/path combination from the request.
// Returns true (permission granted) or false (permission forbidden)
func (a *BasicAuthorizer) CheckPermission(r *http.Request) bool {
	user := a.GetUserName(r)
	method := r.Method
	path := r.URL.Path
	return a.enforcer.Enforce(user, path, method)
}

// RequirePermission returns the 403 Forbidden to the client
func (a *BasicAuthorizer) RequirePermission(c *core.Context) {
	c.Abort(403, "Forbidden")
}
