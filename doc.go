/*
Package replay enables creating and/or replaying canned HTTP server responses.

It is primarily intended to simplify providing canned HTTP responses for unit
testing packages that rely on external HTTP services. It may have other uses.

Recordings can be created by either using the HTTP client returned by
NewRecordingClient, or by setting the Recording field of the RoundTripper to
true. The recording format is intended to be simple, so that they may also be
created and edited manually for simple cases.

Recordings are identified uniquely by a pathname derived from HTTP request
contents. Paths are constructed as a series of intermediate directories and a
file as follows:
	HTTP scheme / host(:port) / HTTP method / path / ... / request (. CRC) .json
The period and CRC beteen "request" and ".json" may not be present if a request
contained no excluded query parameters, no excluded headers and also no body.
Each component is also URL-encoded, if necessary, with url.QueryEscape, to avoid
potentially invalid filenames for some platforms. As an example, a GET request
for the URL
	http://www.example.com/path/to/easy+street
generates the path name
	http/www.example.com/GET/path/to/easy%2bstreet/request.json
The RoundTripper will first try to load a canned response from the path with the
CRC extension, if a CRC is calculated. If no response is found, it will by
default attempt to load the content from a path without the CRC extension. This
behavior can be disabled by setting StrictPath to true.

The paths above are relative to the Dir field of RoundTripper, which is also
taken as a parameter to the NewClient and NewRecordingClient functions.

The format of the recording files is also intended to be easily human-readable.
The first part of the file is a JSON object with fields that will be mapped to
the *http.Response object. The JSON object is followed by one newline. Any
content after that is the body of the recorded response:
	{
	  "status": "404 Not Found",
	  "status_code": 301,
	  "proto": "HTTP/1.1",
	  "proto_major": 1,
	  "proto_minor": 1,
	  "headers": {
	    "Content-Type": [
	      "text/plain"
	    ],
	  }
	}
	The requested content was not found.

A simple example use case may look something like this:
	client := replay.NewClient("testdata")
	// If allowRecording is false, this will only succeed if a recorded response
	// exists under the "testdata" directory:
	res, err := client.Get("https://api.ipify.org?format=json")
*/
package replay
