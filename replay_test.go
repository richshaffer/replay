package replay

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordReplay(t *testing.T) {
	require, assert := require.New(t), assert.New(t)
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Custom-Header", "CustomValue")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "not found")
		},
	))
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	client := NewClient(tmpDir)
	res, err := client.Get(server.URL + "/test/path")
	require.NoError(err)
	require.Equal(http.StatusNotFound, res.StatusCode)
	server.Close()

	// Repeat request. Should load from the test file.
	res, err = client.Get(server.URL + "/test/path")
	if assert.NoError(err) && assert.NotNil(res) {
		buf, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		assert.Equal("not found\n", string(buf))
		assert.Equal(http.StatusNotFound, res.StatusCode)
		assert.Equal("CustomValue", res.Header.Get("X-Custom-Header"))
	}
}

func TestHeaders(t *testing.T) {
	require, assert := require.New(t), assert.New(t)
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			json.NewEncoder(w).Encode(req.Header)
		},
	))
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	client := NewClient(tmpDir)
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/test/path", nil)
	req.Header = http.Header{"Header-A": []string{"a"}}
	res, err := client.Do(req)
	require.NoError(err)
	require.Equal(http.StatusOK, res.StatusCode)
	req.Header = http.Header{"Header-B": []string{"b"}}
	res, err = client.Do(req)
	require.NoError(err)
	require.Equal(http.StatusOK, res.StatusCode)
	server.Close()

	req.Header = http.Header{"Header-A": []string{"a"}}
	res, err = client.Do(req)
	if assert.NoError(err) && assert.NotNil(res) {
		var body http.Header
		assert.NoError(json.NewDecoder(res.Body).Decode(&body))
		res.Body.Close()
		assert.Equal("a", body.Get("Header-A"))
		assert.Empty(body.Get("Header-B"))
		assert.Equal(http.StatusOK, res.StatusCode)
	}

	req.Header = http.Header{"Header-B": []string{"b"}}
	res, err = client.Do(req)
	if assert.NoError(err) && assert.NotNil(res) {
		var body http.Header
		assert.NoError(json.NewDecoder(res.Body).Decode(&body))
		res.Body.Close()
		assert.Empty(body.Get("Header-A"))
		assert.Equal("b", body.Get("Header-B"))
		assert.Equal(http.StatusOK, res.StatusCode)
	}
}

func TestQueryString(t *testing.T) {
	require, assert := require.New(t), assert.New(t)
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			json.NewEncoder(w).Encode(req.URL.Query())
		},
	))
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	queryA := url.Values{"querya": []string{"a"}}
	queryB := url.Values{"queryb": []string{"b"}}

	client := NewClient(tmpDir)
	res, err := client.Get(server.URL + "?" + queryA.Encode())
	require.NoError(err)
	require.Equal(http.StatusOK, res.StatusCode)
	res, err = client.Get(server.URL + "?" + queryB.Encode())
	require.NoError(err)
	require.Equal(http.StatusOK, res.StatusCode)
	server.Close()

	res, err = client.Get(server.URL + "?" + queryA.Encode())
	if assert.NoError(err) && assert.NotNil(res) {
		var body url.Values
		assert.NoError(json.NewDecoder(res.Body).Decode(&body))
		res.Body.Close()
		assert.Equal("a", body.Get("querya"))
		assert.Empty(body.Get("queryb"))
		assert.Equal(http.StatusOK, res.StatusCode)
	}

	res, err = client.Get(server.URL + "?" + queryB.Encode())
	if assert.NoError(err) && assert.NotNil(res) {
		var body url.Values
		assert.NoError(json.NewDecoder(res.Body).Decode(&body))
		res.Body.Close()
		assert.Empty(body.Get("querya"))
		assert.Equal("b", body.Get("queryb"))
		assert.Equal(http.StatusOK, res.StatusCode)
	}
}

func TestMethod(t *testing.T) {
	require, assert := require.New(t), assert.New(t)
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			io.Copy(w, req.Body)
		},
	))
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	client := NewClient(tmpDir)
	req, _ := http.NewRequest(
		http.MethodPost, server.URL+"/", strings.NewReader("bodya"),
	)
	res, err := client.Do(req)
	require.NoError(err)
	require.Equal(http.StatusOK, res.StatusCode)
	req, _ = http.NewRequest(
		http.MethodPut, server.URL+"/", strings.NewReader("bodyb"),
	)
	res, err = client.Do(req)
	require.NoError(err)
	require.Equal(http.StatusOK, res.StatusCode)
	server.Close()

	req, _ = http.NewRequest(
		http.MethodPost, server.URL+"/", strings.NewReader("bodya"),
	)
	res, err = client.Do(req)
	if assert.NoError(err) && assert.NotNil(res) {
		buf, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		assert.Equal("bodya", string(buf))
		assert.Equal(http.StatusOK, res.StatusCode)
	}

	req, _ = http.NewRequest(
		http.MethodPut, server.URL+"/", strings.NewReader("bodyb"),
	)
	res, err = client.Do(req)
	if assert.NoError(err) && assert.NotNil(res) {
		buf, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		assert.Equal("bodyb", string(buf))
		assert.Equal(http.StatusOK, res.StatusCode)
	}
}
