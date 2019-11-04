package replay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

// A Recording represents a recorded HTTP server response. The fields map
// directly to fields in http.Response, except for Body, which is the body of
// the server response.
type Recording struct {
	Status     string      `json:"status,omitempty"`
	StatusCode int         `json:"status_code,omitempty"`
	Proto      string      `json:"proto,omitempty"`
	ProtoMajor int         `json:"proto_major,omitempty"`
	ProtoMinor int         `json:"proto_minor,omitempty"`
	Headers    http.Header `json:"headers,omitempty"`
	Body       []byte      `json:"-"`
}

// NewRecording returns a new, populated Recording struct from the given
// *http.Response. The http.Response Body is read and replaced.
func NewRecording(res *http.Response) (*Recording, error) {
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	res.Body = ioutil.NopCloser(bytes.NewReader(body))

	rec := &Recording{
		Status:     res.Status,
		StatusCode: res.StatusCode,
		Proto:      res.Proto,
		ProtoMajor: res.ProtoMajor,
		ProtoMinor: res.ProtoMinor,
		Headers:    res.Header,
		Body:       body,
	}

	return rec, nil
}

// LoadRecording loads a Recording object from the given file path.
func LoadRecording(path string) (*Recording, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rec *Recording
	dec := json.NewDecoder(f)
	if err = dec.Decode(&rec); err != nil {
		return nil, err
	}
	// dec.Buffered() is a bytes.Reader around the []byte buffered in Decoder.
	// It isn't all of the data in f.
	r := bufio.NewReader(io.MultiReader(dec.Buffered(), f))
	// Encode writes a trailing newline, but Decode doesn't parse it.
	if buf, err := r.Peek(1); err == nil && buf[0] == '\n' {
		r.ReadByte()
	}
	if rec.Body, err = ioutil.ReadAll(r); err != nil {
		return nil, err
	}
	return rec, nil
}

// Save writes the Recording to the given path. The file is written to a
// temporary file and then renamed to ensure atomicity.
func (r *Recording) Save(path string) error {
	dir, filename := filepath.Split(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	f, err := ioutil.TempFile(dir, filename)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(&r); err == nil {
		_, err = f.Write(r.Body)
	}
	f.Close()
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

// Response returns an *http.Response object from the populated Recording.
func (r *Recording) Response() *http.Response {
	return &http.Response{
		Status:     r.Status,
		StatusCode: r.StatusCode,
		Proto:      r.Proto,
		ProtoMajor: r.ProtoMajor,
		ProtoMinor: r.ProtoMinor,
		Header:     r.Headers,
		Body:       ioutil.NopCloser(bytes.NewReader(r.Body)),
	}
}
