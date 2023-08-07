package test

import (
	"bytes"
	"io"
	"net/http"
)

func MockHttpClientReturn(success bool, body string) (*http.Response, error) {
	if success {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(body))}, nil
	}
	return &http.Response{StatusCode: 400, Body: io.NopCloser(bytes.NewBufferString(body))}, nil
}
