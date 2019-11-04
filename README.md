# Replay

Replay is a package for easily recording and replaying HTTP responses, e.g. for
using canned API responses for unit testing.

This work is licensed under the ISC License, a copy of which can be found at [LICENSE](LICENSE)

Installation
------------

```bash
go get -u github.com/richshaffer/replay
```

Documentation
-------------
https://godoc.org/github.com/richshaffer/replay

Example
-------

```go
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/richshaffer/replay"
)

func main() {
	// If false, only already-recorded requests will succeed:
	record := flag.Bool("record", false, "enable recording new API responses")
	flag.Parse()

	client := replay.NewClient("testdata")
	client.Transport.(*replay.RoundTripper).OmitHeaders.Add("X-Amz-Date")

	config := aws.NewConfig().WithHTTPClient(client)
	sess := session.Must(session.NewSession(config))
	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("testkey"),
		Body:   strings.NewReader("testbody"),
	})
	if err != nil {
		fmt.Printf("failed to upload file, %v\n", err)
		return
	}
	fmt.Printf("file uploaded to, %s\n", result.Location)
}

```
