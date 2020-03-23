package cookie_transcoder

type ICookieTranscoder interface {
	Encode (cookieName string, value interface{}) (string, error)
	Decode (cookieName string, cookieValue string, v interface{}) error
}
