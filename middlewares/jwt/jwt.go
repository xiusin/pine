package jwt

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/xiusin/pine"
	"net/http"
	"strings"
)

type errorHandler func(w http.ResponseWriter, r *http.Request, err string)

type TokenExtractor func(r *http.Request) (string, error)

type JwtOptions struct {
	Secret              []byte
	validationKeyGetter jwt.Keyfunc
	UserProperty        string
	ErrorHandler        errorHandler
	CredentialsOptional bool
	Extractor           TokenExtractor
	Debug               bool
	EnableAuthOnOptions bool
	SigningMethod       jwt.SigningMethod
	DefaultClaims       jwt.Claims
}

type Middleware struct {
	Options JwtOptions
}

func OnError(w http.ResponseWriter, r *http.Request, err string) {
	http.Error(w, err, http.StatusUnauthorized)
}

func NewJwt(opts JwtOptions) *Middleware {
	if opts.Secret == nil {
		panic("place set secret")
	}
	opts.validationKeyGetter = func(token *jwt.Token) (i interface{}, err error) {
		return opts.Secret, nil
	}

	if opts.DefaultClaims == nil {
		opts.DefaultClaims = &jwt.StandardClaims{}
	}

	if opts.UserProperty == "" {
		opts.UserProperty = "user"
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = OnError
	}

	if opts.Extractor == nil {
		opts.Extractor = FromAuthHeader
	}

	return &Middleware{
		Options: opts,
	}
}

func (m *Middleware) logf(format string, args ...interface{}) {
	if m.Options.Debug {
		pine.Logger().Printf(format, args...)
	}
}

func (m *Middleware) Serve() pine.Handler {
	return func(c *pine.Context) {
		c.Set("jwt.secret", m.Options.Secret)
		c.Set("jwt.signingMethod", m.Options.SigningMethod)

		if err := m.CheckJWT(c); err != nil {
			m.Options.ErrorHandler(c.Writer(), c.Request(), err.Error())
			return
		}
		c.Next()
	}
}

func FromAuthHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", nil // No error, just no token
	}

	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", errors.New("Authorization header format must be Bearer {token}")
	}

	return authHeaderParts[1], nil
}

func FromParameter(param string) TokenExtractor {
	return func(r *http.Request) (string, error) {
		return r.URL.Query().Get(param), nil
	}
}

func FromFirst(extractors ...TokenExtractor) TokenExtractor {
	return func(r *http.Request) (string, error) {
		for _, ex := range extractors {
			token, err := ex(r)
			if err != nil {
				return "", err
			}
			if token != "" {
				return token, nil
			}
		}
		return "", nil
	}
}

func (m *Middleware) CheckJWT(c *pine.Context) error {
	r, w := c.Request(), c.Writer()
	if !m.Options.EnableAuthOnOptions {
		if r.Method == "OPTIONS" {
			return nil
		}
	}

	token, err := m.Options.Extractor(c.Request())
	if err != nil {
		m.logf("Error extracting JWT: %v", err)
	} else {
		m.logf("Token extracted: %s", token)
	}
	if err != nil {
		m.Options.ErrorHandler(w, r, err.Error())
		return fmt.Errorf("Error extracting token: %v", err)
	}

	if token == "" {
		if m.Options.CredentialsOptional {
			m.logf("  No credentials found (CredentialsOptional=true)")
			return nil
		}

		errorMsg := "Required authorization token not found"
		m.Options.ErrorHandler(w, r, errorMsg)
		m.logf("  Error: No credentials found (CredentialsOptional=false)")
		return fmt.Errorf(errorMsg)
	}

	parsedToken, err := jwt.ParseWithClaims(token, m.Options.DefaultClaims, m.Options.validationKeyGetter)
	if err != nil {
		m.logf("Error parsing token: %v", err)
		m.Options.ErrorHandler(w, r, err.Error())
		return fmt.Errorf("Error parsing token: %v", err)
	}

	if m.Options.SigningMethod != nil && m.Options.SigningMethod.Alg() != parsedToken.Header["alg"] {
		message := fmt.Sprintf("Expected %s signing method but token specified %s",
			m.Options.SigningMethod.Alg(),
			parsedToken.Header["alg"])
		m.logf("Error validating token algorithm: %s", message)
		m.Options.ErrorHandler(w, r, errors.New(message).Error())
		return fmt.Errorf("Error validating token algorithm: %s", message)
	}

	if !parsedToken.Valid {
		m.logf("Token is invalid")
		m.Options.ErrorHandler(w, r, "The token isn't valid")
		return errors.New("Token is invalid")
	}
	c.Set("jwt.tokenClaims", parsedToken.Claims)
	return nil
}
