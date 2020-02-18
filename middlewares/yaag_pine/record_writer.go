// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package yaag_pine

import (
	"log"
	"net/http"
)

type RecordWriter struct {
	http.ResponseWriter

	status int
	body   []byte
}

func init()  {
	log.Println("yagg_pine RecordWriter will replaced http.ResponseWriter")
}

var _ http.ResponseWriter =  (*RecordWriter)(nil)

func NewWriter(w http.ResponseWriter) *RecordWriter {
	return &RecordWriter{status: http.StatusOK, ResponseWriter: w}
}

func (w *RecordWriter) Write(body []byte) (int, error) {
	w.body = body
	return w.ResponseWriter.Write(body)
}

func (w *RecordWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *RecordWriter) GetBody() []byte {
	return w.body
}

func (w *RecordWriter) GetStatus() int  {
	return w.status
}
