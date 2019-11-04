package replay

import (
	"net/http"
	"os"
	"path/filepath"
)

const (
	// ModeRecordIfMissing enables playing back recordings that exist, and
	// recording new responses when a recording isn't found.
	ModeRecordIfMissing = iota
	// ModePlaybackOnly enables playing back content only.
	ModePlaybackOnly
	// ModeRecordOnly enables recording new content only.
	ModeRecordOnly
)

// RoundTripper implemnts a wrapper around an instance of the http.RoundTripper
// interface type. It attempts toload canned responses from recordings on disk.
// If one is not found, it can also use the wrapped RoundTripper to fetch the
// response and record it to disk for later use.
type RoundTripper struct {
	// RoundTripper is the http.RoundTripper used to process HTTP requests if
	// a recorded response is not available on disk. It will be unused if
	// Record is false.
	http.RoundTripper
	// Dir is the base directory where HTTP responses are read from and recored
	// to.
	Dir string
	// Mode determines if responses are recorded, played back, or recorded only
	// if missing.
	Mode int
	// PathGenerator is used to generate unique paths for retrieving and saving
	// responses. The paths generated are relative to Dir.
	*PathGenerator
	// StrictPath, if true, will prevent RoundTripper from loading responses
	// without a checksum. The default is to attempt to load a recording from
	// the path without a checksum in cases where the path including the
	// checksum does not exist.
	StrictPath bool
}

// RoundTrip wraps the underyling RoundTrip implementation in order to enable
// loading or recording HTTP server responses.
func (r *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recordingPath, err := r.PathGenerator.RecordingPath(req)
	if err != nil {
		return nil, &Error{Request: req, Err: err}
	}

	path := filepath.Join(r.Dir, recordingPath.Path())
	genericPath := filepath.Join(r.Dir, recordingPath.GenericPath())

	if r.Mode != ModeRecordOnly {
		rec, err := LoadRecording(path)
		if !r.StrictPath && genericPath != path && os.IsNotExist(err) {
			rec, err = LoadRecording(genericPath)
		}
		if err == nil {
			return rec.Response(), nil
		}
		if r.Mode == ModePlaybackOnly || !os.IsNotExist(err) {
			return nil, err
		}
	}

	res, err := r.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	rec, err := NewRecording(res)
	if err != nil {
		return nil, &Error{Request: req, Response: res, Err: err}
	}
	if err = rec.Save(path); err != nil {
		return nil, &Error{Request: req, Response: res, Err: err}
	}
	return res, err
}

// NewClient returns an *http.Client which will return pre-recorded responses if
// the exists, or create new recordings if they are missing..
func NewClient(dir string) *http.Client {
	return &http.Client{
		Transport: &RoundTripper{
			Dir:           dir,
			RoundTripper:  http.DefaultTransport,
			PathGenerator: NewPathGenerator(),
		},
	}
}

// NewPlaybackOnlyClient returns an *http.Client which will only return pre-
// recorded responses. If no response is found, an error is returned.
func NewPlaybackOnlyClient(dir string) *http.Client {
	client := NewClient(dir)
	client.Transport.(*RoundTripper).Mode = ModePlaybackOnly
	return client
}

// NewRecordOnlyClient returns an *http.Client which will record new responses,
// even if a pre-recorded response exists.
func NewRecordOnlyClient(dir string) *http.Client {
	client := NewClient(dir)
	client.Transport.(*RoundTripper).Mode = ModeRecordOnly
	return client
}
