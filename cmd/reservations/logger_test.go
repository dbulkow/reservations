/* Copyright (c) 2021 David Bulkow */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type logtest struct {
	code    int
	content string
}

func (lt *logtest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", lt.content)
	w.WriteHeader(lt.code)
	fmt.Fprintf(w, "response text")
}

func TestLogger(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "path/to/file", nil)
	r.Header.Set("User-Agent", "test")
	r.RemoteAddr = "123.456.789.012"
	w := httptest.NewRecorder()
	handler := logger(&logtest{code: http.StatusOK, content: "text/plain"})
	handler.ServeHTTP(w, r)
}

func TestLoggerJSON(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "path/to/file", nil)
	r.Header.Set("User-Agent", "test")
	r.RemoteAddr = "123.456.789.012"
	w := httptest.NewRecorder()
	handler := logger(&logtest{code: http.StatusOK, content: "application/json"})
	handler.ServeHTTP(w, r)
}

func TestLoggerError(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "path/to/file", nil)
	r.Header.Set("User-Agent", "test")
	r.RemoteAddr = "123.456.789.012"
	w := httptest.NewRecorder()
	handler := logger(&logtest{code: http.StatusNotFound, content: "application/json"})
	handler.ServeHTTP(w, r)
}

func TestLoggerRequestBody(t *testing.T) {
	data := struct {
		Status string
		Value  string
	}{
		Status: "Success",
		Value:  "some value",
	}
	body, _ := json.Marshal(&data)
	b := bytes.NewBuffer(body)

	r, _ := http.NewRequest(http.MethodPost, "path/to/file", b)
	r.Header.Set("User-Agent", "test")
	r.RemoteAddr = "123.456.789.012"
	w := httptest.NewRecorder()
	handler := logger(&logtest{code: http.StatusNotFound, content: "application/json"})
	handler.ServeHTTP(w, r)
}
