//go:build dart

package http

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"strings"
)

// defaultMaxMemory is the count of bytes of a multipart form that stays in
// memory. Go holds the same value.
const defaultMaxMemory = 32 << 20

// ErrNotMultipart reports a body whose type is not multipart/form-data.
var ErrNotMultipart = errors.New("request Content-Type isn't multipart/form-data")

// ErrMissingBoundary reports a header of the type multipart/form-data with no
// boundary.
var ErrMissingBoundary = errors.New("no multipart boundary param in Content-Type")

// ParseForm reads the query of the URL and the body of a form into Form, and it
// reads the body alone into PostForm.
//
// A body reaches PostForm only when the method is POST, PUT, or PATCH, and only
// when the type of the body is application/x-www-form-urlencoded.
func (r *Request) ParseForm() error {
	if r.PostForm == nil {
		if r.Method == MethodPost || r.Method == MethodPut || r.Method == MethodPatch {
			r.PostForm = url.Values{}
			values, err := r.parseBodyForm()
			if err != nil {
				return err
			}
			r.PostForm = values
		} else {
			r.PostForm = url.Values{}
		}
	}
	if r.Form != nil {
		return nil
	}
	r.Form = url.Values{}
	for key, values := range r.PostForm {
		r.Form[key] = append(r.Form[key], values...)
	}
	if r.URL == nil {
		return nil
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	for key, values := range query {
		r.Form[key] = append(r.Form[key], values...)
	}
	return err
}

func (r *Request) parseBodyForm() (url.Values, error) {
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || kind != "application/x-www-form-urlencoded" {
		return url.Values{}, nil
	}
	if r.Body == nil {
		return url.Values{}, errors.New("missing form body")
	}
	text, err := io.ReadAll(r.Body)
	if err != nil {
		return url.Values{}, err
	}
	return url.ParseQuery(string(text))
}

// MultipartReader gives a reader of the parts of a multipart body. A caller
// that reads the parts itself avoids the memory of ParseMultipartForm.
func (r *Request) MultipartReader() (*multipart.Reader, error) {
	if r.MultipartForm != nil {
		return nil, errors.New("http: MultipartReader called twice")
	}
	return r.multipartReader()
}

func (r *Request) multipartReader() (*multipart.Reader, error) {
	line := r.Header.Get("Content-Type")
	if line == "" {
		return nil, ErrNotMultipart
	}
	kind, params, err := mime.ParseMediaType(line)
	if err != nil || !strings.HasPrefix(kind, "multipart/") {
		return nil, ErrNotMultipart
	}
	boundary, ok := params["boundary"]
	if !ok {
		return nil, ErrMissingBoundary
	}
	if r.Body == nil {
		return nil, errors.New("missing form body")
	}
	return multipart.NewReader(r.Body, boundary), nil
}

// ParseMultipartForm reads a body of the type multipart/form-data into
// MultipartForm. A part that is larger than maxMemory goes to a file of the
// temporary directory.
//
// It also calls ParseForm, and each value of the form joins Form.
func (r *Request) ParseMultipartForm(maxMemory int64) error {
	if r.MultipartForm != nil {
		return nil
	}
	reader, err := r.multipartReader()
	if err != nil {
		return err
	}
	form, err := reader.ReadForm(maxMemory)
	if err != nil {
		return err
	}
	if r.PostForm == nil {
		r.PostForm = url.Values{}
	}
	if r.Form == nil {
		r.Form = url.Values{}
		if r.URL != nil {
			query, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil {
				return err
			}
			for key, values := range query {
				r.Form[key] = append(r.Form[key], values...)
			}
		}
	}
	for key, values := range form.Value {
		r.Form[key] = append(r.Form[key], values...)
		r.PostForm[key] = append(r.PostForm[key], values...)
	}
	r.MultipartForm = form
	return nil
}

// FormValue gives the first value of a key of Form. It calls ParseMultipartForm
// and ParseForm, and it drops each error of the 2.
func (r *Request) FormValue(key string) string {
	if r.Form == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	if values := r.Form[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// PostFormValue gives the first value of a key of PostForm.
func (r *Request) PostFormValue(key string) string {
	if r.PostForm == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	if values := r.PostForm[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// FormFile gives the first file of a key of MultipartForm.
func (r *Request) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
			return nil, nil, err
		}
	}
	if headers := r.MultipartForm.File[key]; len(headers) > 0 {
		file, err := headers[0].Open()
		return file, headers[0], err
	}
	return nil, nil, ErrMissingFile
}

// ErrMissingFile reports a key that the form holds with no file.
var ErrMissingFile = errors.New("http: no such file")
