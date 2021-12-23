package cookie_transcoder

type AbstractCookieTranscoder interface {
	Encode(cookieName string, value interface{}) (string, error)
	Decode(cookieName string, cookieValue string, v interface{}) error
}
