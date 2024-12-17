package contracts

type CookieTranscoder interface {
	Encode(string, any) (string, error)
	Decode(string, string, any) error
}
