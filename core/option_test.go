package core

import "testing"

var testOption = DefaultOptions()

func TestOption_Get(t *testing.T) {
	testOption.MergeOption(&Option{
		Others: map[string]interface{}{
			"age": 29,
		},
	})
	t.Log(testOption.Get("age"))
}

func TestOption_Add(t *testing.T) {
	t.Log(testOption.Add("name","xiusin"))
}


