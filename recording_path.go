package replay

import (
	"bytes"
	"hash"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// StringSet implements a set of string values.
type StringSet map[string]struct{}

// NewStringSet returns a new set initialized with optional values.
func NewStringSet(args ...string) StringSet {
	ss := make(StringSet, len(args))
	ss.Add(args...)
	return ss
}

// Add adds the provided value(s) to the set.
func (ss StringSet) Add(args ...string) {
	for i := range args {
		ss[args[i]] = struct{}{}
	}
}

// Del removes the provided value(s) from the set.
func (ss StringSet) Del(args ...string) {
	for i := range args {
		delete(ss, args[i])
	}
}

// DefaultOmitHeaders returns a default set of headers to omit from recording
// path generation.
func DefaultOmitHeaders() StringSet {
	return StringSet{
		"Authorization":       struct{}{},
		"Connection":          struct{}{},
		"Date":                struct{}{},
		"Proxy-Authorization": struct{}{},
		"Transfer-Encoding":   struct{}{},
		"Upgrade":             struct{}{},
	}
}

// RecordingPath contains a relative path for a recording.
type RecordingPath struct {
	dir      string
	checksum string
}

// Path returns a canonical filename generated for the request. If a checksum
// can be calculated over the request query parameters, headers and body,
// the filename portion of the path will be "recording." + checksum + "".json".
// If there are no query parameters, no headers and no body, the returned path
// will be GenericPath().
func (r *RecordingPath) Path() string {
	if r.checksum != "" {
		return filepath.Join(r.dir, "request."+r.checksum+".json")
	}
	return r.GenericPath()
}

// GenericPath returns a generic path for the request. The filename portion is
// always "request.json", even if a checksum was calculated.
func (r *RecordingPath) GenericPath() string {
	return filepath.Join(r.dir, "request.json")
}

// PathGenerator creates a unique path for a given *http.Request.
type PathGenerator struct {
	// OmitHeaders is a set of headers to exclude from path calculations.
	// Requests with different content in these headers can still return the
	// same unique path.
	OmitHeaders StringSet
	// OmitQuery is a set of query parameters to exclude from path calculations.
	// Requests with different content in these parameters can still return the
	// same unique path.
	OmitQuery StringSet
	// MungeRequestBody can be used to edit which bytes of the request body
	// are used to calculate the path CRC. It may be nil or return the same
	// io.Reader that is passed in. It does not alter the request that is sent
	// to the server.
	MungeRequestBody func(*http.Request, io.Reader) io.Reader
}

// NewPathGenerator creates a new generator for recording path names.
func NewPathGenerator() *PathGenerator {
	return &PathGenerator{OmitHeaders: DefaultOmitHeaders()}
}

// RecordingPath returns the unique path for the given request.
func (p *PathGenerator) RecordingPath(req *http.Request) (*RecordingPath, error) {
	parts := make([]string, 0, 10)
	if req.URL.Scheme != "" {
		parts = append(parts, req.URL.Scheme)
	}
	if req.URL.Host != "" {
		parts = append(parts, url.QueryEscape(req.URL.Host))
	}
	if req.Method != "" {
		parts = append(parts, req.Method)
	}
	for _, part := range strings.Split(req.URL.Path, "/") {
		// Use QueryEscape, since it captures things like ':' that might not be
		// valid in a path, depending on OS.
		if part != "" {
			parts = append(parts, url.QueryEscape(part))
		}
	}

	crc, err := p.RequestCRC(req)
	if err != nil {
		return nil, err
	}

	path := &RecordingPath{
		dir:      strings.Join(parts, string(os.PathSeparator)),
		checksum: crc,
	}

	return path, nil
}

type hashableMap map[string][]string

func (m hashableMap) updateHash(h hash.Hash, excludes StringSet) bool {
	values := make(sort.StringSlice, 0, len(m))
	for k := range m {
		if _, ok := excludes[k]; !ok {
			values = append(values, k)
		}
	}
	sort.Sort(values)
	for _, k := range values {
		h.Write([]byte(k))
		for _, v := range m[k] {
			h.Write([]byte(v))
		}
	}
	return len(values) > 0
}

// RequestCRC generates a checksum based on the contents of any headers, query
// string parameters and body in the request. Any headers in OmitHeaders or any
// query string parameters in OmitQuery are not considered. If there are no
// headers, query string parameters and body to consider, returns an empty
// string.
func (p *PathGenerator) RequestCRC(req *http.Request) (string, error) {
	q := req.URL.Query()
	h := crc32.NewIEEE()
	hasHash := hashableMap(q).updateHash(h, p.OmitQuery)
	hasHash = hashableMap(req.Header).updateHash(h, p.OmitHeaders) || hasHash

	if req.Body != nil {
		if _, ok := req.Body.(io.ReadSeeker); !ok && req.GetBody == nil {
			body, err := ioutil.ReadAll(req.Body)
			req.Body.Close()
			if err != nil {
				return "", err
			}
			req.Body = ioutil.NopCloser(bytes.NewReader(body))
		}

		var r io.Reader = req.Body
		if p.MungeRequestBody != nil {
			r = p.MungeRequestBody(req, req.Body)
		}
		n, err := io.Copy(h, r)
		if seeker, ok := req.Body.(io.Seeker); ok {
			if err == nil {
				_, err = seeker.Seek(io.SeekStart, 0)
			}
			if err != nil {
				req.Body.Close()
				return "", err
			}
		} else {
			req.Body.Close()
			if req.Body, err = req.GetBody(); err != nil {
				return "", err
			}
		}
		hasHash = hasHash || n > 0
	}

	if hasHash {
		return strconv.FormatUint(uint64(h.Sum32()), 10), nil
	}
	return "", nil
}
