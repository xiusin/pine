// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
)

type Template struct{}

func (_ *Template) JSON(writer io.Writer, v map[string]interface{}) error {
	b, err := json.Marshal(v)
	if err == nil {
		_, err = writer.Write(b)
	}
	return err
}

func (_ *Template) JSONP(writer io.Writer, callback string, v map[string]interface{}) error {
	var ret bytes.Buffer
	b, err := json.Marshal(v)
	if err == nil {
		ret.Write([]byte(callback))
		ret.Write([]byte("("))
		ret.Write(b)
		ret.Write([]byte(")"))
		_, err = writer.Write(ret.Bytes())
	}
	return err
}

func (_ *Template) Text(writer io.Writer, v []byte) error {
	_, err := writer.Write(v)
	return err
}

//todo don't support now
func (_ *Template) XML(writer io.Writer, v map[string]interface{}) error {
	b, err := xml.MarshalIndent(v, "", " ")
	if err == nil {
		_, err = writer.Write(b)
	}
	return err
}
