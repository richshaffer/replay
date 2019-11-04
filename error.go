package replay

import "net/http"

// Error is an error that may be returned by RoundTripper, and thus by the
// *http.Client returned by NewClient or NewRecordingClient. It can be used to
// differentiate an error encountered when trying to fetch or save a recording
// in local storage versus errors returned by the http package, such as for URL
// or network errors.
type Error struct {
	// Request is the *http.Request that was being processed when the error
	// occurred.
	Request *http.Request
	// Response is the *http.Response that was being processed when the error
	// occurred. It will be nil if the error occurred while loading a recording
	// from the local filesystem (as opposed to saving a new response).
	Response *http.Response
	// Err is the underlying error that was encountered while attempting to
	// manipulate a recording.
	Err error
}

func (r *Error) Error() string {
	return r.Err.Error()
}
