package helper

import (
	"testing"
)

type MyCustom struct {
}

func TestGetTypeName(t *testing.T) {
	t.Log(GetTypeName(new(MyCustom)))
}
