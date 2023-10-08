package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var bufPool sync.Pool

func getBuf() *bytes.Buffer {
	buf := bufPool.Get()
	if buf == nil {
		return &bytes.Buffer{}
	}
	return buf.(*bytes.Buffer)
}

func giveBuf(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}

// decorateWriter wraps a writer to handle gzip compression if requested.  It
// returns the decorated writer and the appropriate "Content-Encoding" header
// (which is empty if no compression is enabled).
func decorateWriter(request *http.Request, writer io.Writer) (w io.Writer, encoding string) {
	header := request.Header.Get(acceptEncodingHeader)
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part := strings.TrimSpace(part)
		if part == "gzip" || strings.HasPrefix(part, "gzip;") {
			return gzip.NewWriter(writer), "gzip"
		}
	}
	return writer, ""
}

// LogFunc is an adapter to allow the use of any function as a promhttp.Logger. If f is a function, LogFunc(f) is a
// promhttp.Logger that calls f.
type LogFunc func(args ...interface{})

// Println implements promhttp.Logger.
func (log LogFunc) Println(args ...interface{}) {
	log(args)
}

// OfBool returns bool address.
func OfBool(i bool) *bool {
	return &i
}
