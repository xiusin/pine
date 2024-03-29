//go:build go1.18
// +build go1.18

package pointer

func To[T interface{}](t T) *T {
	return &t
}

func ToOrNil[T comparable](t T) *T {
	if z, ok := interface{}(t).(interface{ IsZero() bool }); ok {
		if z.IsZero() {
			return nil
		}
		return &t
	}

	var zero T
	if t == zero {
		return nil
	}
	return &t
}

func Get[T interface{}](t *T) T {
	if t == nil {
		var zero T
		return zero
	}
	return *t
}
