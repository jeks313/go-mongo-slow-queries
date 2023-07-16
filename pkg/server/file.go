package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
)

// File serves up a file, optionally transforming it with a TransformFunc
func File(r *mux.Router, filename string, b *bytes.Buffer) {
	r.Handle(filename, fileHandler(b))
}

func fileHandler(b *bytes.Buffer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		w.Write(b.Bytes())
	})
}

// FormatFunc functions are used to apply transformations to file data, such as applying templates
type FormatFunc func(io.Reader) io.Reader

// FormatMarkdown takes a markdown file and applies the js template to auto-render it
func FormatMarkdown(f io.Reader) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte(""))
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		slog.Error("failed to read input for formatting", "error", err)
		return buf
	}
	header := `<!DOCTYPE html><html><title>Docs</title><xmp theme="cerulean" style="display:block;">`
	footer := `</xmp><script src="http://strapdownjs.com/v/0.2/strapdown.js"></script></html>`
	buf.WriteString(fmt.Sprintf("%s\n%s\n%s\n", header, contents, footer))
	return buf
}
