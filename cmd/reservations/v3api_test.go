/* Copyright (c) 2021 David Bulkow */

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"testing"
	"time"

	. "github.com/dbulkow/reservations/api"
)

type apiStorage struct {
	error        error
	reservations []*Reservation
}

func (s *apiStorage) GetById(resid int) (*Reservation, error) {
	if len(s.reservations) == 0 {
		return nil, s.error
	}

	return s.reservations[0], s.error
}

func (s *apiStorage) List(resource, show string, start, length int) ([]*Reservation, error) {
	if s.error != nil {
		return nil, s.error
	}

	res := make([]*Reservation, 0)

	if length == 0 || length > len(s.reservations) {
		length = len(s.reservations)
	}

	for i := 0; i < length; i++ {
		res = append(res, s.reservations[i])
	}

	return res, nil
}

func (s *apiStorage) Add(res *Reservation) error {
	res.LastModified = time.Now()
	return s.error
}

func (s *apiStorage) Update(ref int, res *Reservation) (*Reservation, error) {
	res.LastModified = time.Now()
	return res, s.error
}

func (s *apiStorage) Delete(ref int, last time.Time) error { return s.error }

type badReader struct{}

func (r *badReader) Read([]byte) (int, error) { return 0, errors.New("fail") }

func TestV3APIOptions(t *testing.T) {
	handler := v3res(&apiStorage{})
	req, _ := http.NewRequest(http.MethodOptions, "*", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 405 got %d", resp.StatusCode)
	}

	exp := "text/plain; charset=utf-8"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}

	exp = "OPTIONS, HEAD, GET, POST"
	if resp.Header.Get("Allow") != exp {
		t.Fatalf("expected allow field \"%s\" got \"%s\"", exp, resp.Header.Get("Allow"))
	}

	req, _ = http.NewRequest(http.MethodOptions, "42", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	resp = w.Result()

	out, err = httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 405 got %d", resp.StatusCode)
	}

	exp = "text/plain; charset=utf-8"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}

	exp = "OPTIONS, HEAD, GET, POST, PUT, PATCH, DELETE"
	if resp.Header.Get("Allow") != exp {
		t.Fatalf("expected allow field \"%s\" got \"%s\"", exp, resp.Header.Get("Allow"))
	}

	exp = "application/json-patch+json, application/merge-patch+json"
	if resp.Header.Get("Accept-Patch") != exp {
		t.Fatalf("expected accept patch \"%s\" got \"%s\"", exp, resp.Header.Get("Accept-Patch"))
	}
}

func TestV3APIMethodNotAllowed(t *testing.T) {
	handler := v3res(&apiStorage{})
	req, _ := http.NewRequest(http.MethodConnect, "", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status code 405 got %d", resp.StatusCode)
	}

	exp := "text/plain; charset=utf-8"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIGet(t *testing.T) {
	now := time.Now()

	storage := &apiStorage{
		reservations: []*Reservation{
			&Reservation{
				ID:           35,
				LastModified: now,
				Resource:     "some resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
			&Reservation{
				ID:           37,
				LastModified: now,
				Resource:     "some other resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
		},
	}

	service, _ = url.Parse("http://localhost")

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIGetCached(t *testing.T) {
	now := time.Now()

	storage := &apiStorage{
		reservations: []*Reservation{
			&Reservation{
				ID:           35,
				LastModified: now,
				Resource:     "some resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
			&Reservation{
				ID:           37,
				LastModified: now,
				Resource:     "some other resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
		},
	}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodGet, "", nil)
	r.Header.Set("If-Modified-Since", now.Format(time.RFC1123))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, false)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("expected status code 304 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIGetLimit(t *testing.T) {
	now := time.Now()

	storage := &apiStorage{
		reservations: []*Reservation{
			&Reservation{
				ID:           35,
				LastModified: now,
				Resource:     "some resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
			&Reservation{
				ID:           37,
				LastModified: now,
				Resource:     "some other resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
			&Reservation{
				ID:           38,
				LastModified: now,
				Resource:     "res3",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
			&Reservation{
				ID:           39,
				LastModified: now,
				Resource:     "res4",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
		},
	}

	u, err := url.Parse("")
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Add("n", "2")

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIGetRef(t *testing.T) {
	now := time.Now()

	storage := &apiStorage{
		reservations: []*Reservation{
			&Reservation{
				ID:           35,
				LastModified: now,
				Resource:     "a thing",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
				Name:         "Some User",
			},
		},
	}

	handler := v3res(storage)
	req, _ := http.NewRequest(http.MethodGet, "35", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIGetRefCached(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:           35,
		LastModified: now,
		Resource:     "a thing",
		Start:        now.Add(30 * time.Second),
		End:          now.Add(60 * time.Second),
		Name:         "Some User",
	}

	storage := &apiStorage{reservations: []*Reservation{res}}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodGet, "35", nil)
	r.Header.Set("If-Modified-Since", res.LastModified.Format(time.RFC1123))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, false)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("expected status code 304 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIGetRefFail(t *testing.T) {
	handler := v3res(&apiStorage{error: errors.New("no data")})
	req, _ := http.NewRequest(http.MethodGet, "0", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 404 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIGetListError(t *testing.T) {
	handler := v3res(&apiStorage{error: errors.New("something broke")})
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code 500 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APINoMatch(t *testing.T) {
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodGet, "abcd", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 404 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APINaN(t *testing.T) {
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodGet, "999999999999999999999999999999999999999999", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 404 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIHead(t *testing.T) {
	now := time.Now()

	storage := &apiStorage{
		reservations: []*Reservation{
			&Reservation{
				ID:           35,
				LastModified: now,
				Resource:     "some resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
			&Reservation{
				ID:           37,
				LastModified: now,
				Resource:     "some other resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
		},
	}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodHead, "", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, false)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIHeadCached(t *testing.T) {
	now := time.Now()

	storage := &apiStorage{
		reservations: []*Reservation{
			&Reservation{
				ID:           35,
				LastModified: now,
				Resource:     "some resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
			&Reservation{
				ID:           37,
				LastModified: now,
				Resource:     "some other resource",
				Start:        now.Add(30 * time.Second),
				End:          now.Add(60 * time.Second),
			},
		},
	}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodHead, "", nil)
	r.Header.Set("If-Modified-Since", now.Format(time.RFC1123))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, false)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("expected status code 304 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIHeadRef(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:           35,
		LastModified: now,
		Resource:     "some resource",
		Start:        now.Add(30 * time.Second),
		End:          now.Add(60 * time.Second),
	}

	storage := &apiStorage{reservations: []*Reservation{res}}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodHead, "35", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, false)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIHeadRefCached(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:           35,
		LastModified: now,
		Resource:     "some resource",
		Start:        now.Add(30 * time.Second),
		End:          now.Add(60 * time.Second),
	}

	storage := &apiStorage{reservations: []*Reservation{res}}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodHead, "35", nil)
	r.Header.Set("If-Modified-Since", now.Format(time.RFC1123))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, false)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("expected status code 304 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIHeadRefNotFound(t *testing.T) {
	handler := v3res(&apiStorage{error: errors.New("not found")})
	r, _ := http.NewRequest(http.MethodHead, "35", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 404 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPost(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		Resource: "thing",
		Start:    now,
		End:      now,
		Name:     "Some User",
		// Email:    "some.user@company.com",
		Initials: "SU",
	}

	resreq, _ := json.MarshalIndent(res, "", "    ")
	b := bytes.NewBuffer(resreq)

	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPost, "", b)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Length", strconv.Itoa(len(resreq)))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code 201 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPostContentLengthInvalid(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		Resource: "thing",
		Start:    now,
		End:      now,
		Name:     "Some User",
		// Email:    "some.user@company.com",
		Initials: "SU",
	}

	resreq, _ := json.MarshalIndent(res, "", "    ")
	b := bytes.NewBuffer(resreq)

	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPost, "", b)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Length", "frank")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code 201 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPostContentLengthTooBig(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		Resource: "thing",
		Start:    now,
		End:      now,
		Name:     "Some User",
		// Email:    "some.user@company.com",
		Initials: "SU",
	}

	resreq, _ := json.MarshalIndent(res, "", "    ")
	b := bytes.NewBuffer(resreq)

	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPost, "", b)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Length", strconv.Itoa(v3MaxRead+1))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code 201 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPostFail(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		Resource: "thing",
		Start:    now,
		End:      now.Add(30 * time.Second),
		Name:     "Some User",
		// Email:    "some.user@company.com",
		Initials: "SU",
	}

	resreq, _ := json.MarshalIndent(res, "", "    ")
	b := bytes.NewBuffer(resreq)

	handler := v3res(&apiStorage{error: errors.New("overlap dude")})
	r, _ := http.NewRequest(http.MethodPost, "", b)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code 400 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPostBadJSON(t *testing.T) {
	b := bytes.NewBufferString("this isn't json")
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPost, "", b)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code 400 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPostWithRef(t *testing.T) {
	b := bytes.NewBufferString("this isn't json")
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPost, "44", b)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status code 405 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPostNotJSON(t *testing.T) {
	b := bytes.NewBufferString("this isn't json")
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPost, "", b)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status code 415 got %s", resp.Status)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIDelete(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:       45,
		Resource: "some resource",
		Start:    now.Add(30 * time.Second),
		End:      now.Add(60 * time.Second),
	}

	storage := &apiStorage{reservations: []*Reservation{res}}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodDelete, strconv.Itoa(res.ID), nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 got %d", resp.StatusCode)
	}
}

func TestV3APIDeleteNotFound(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:       45,
		Resource: "some resource",
		Start:    now.Add(30 * time.Second),
		End:      now.Add(60 * time.Second),
	}

	storage := &apiStorage{reservations: []*Reservation{res}, error: errors.New("not found")}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodDelete, "0", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 400 got %d", resp.StatusCode)
	}
}

func TestV3APIDeleteError(t *testing.T) {
	handler := v3res(&apiStorage{error: errors.New("it broke")})
	r, _ := http.NewRequest(http.MethodDelete, "0", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code 500 got %d", resp.StatusCode)
	}
}

func TestV3APIDeleteNoRef(t *testing.T) {
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodDelete, "", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 400 got %d", resp.StatusCode)
	}
}

func TestV3APIPut(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:       45,
		Resource: "some resource",
		Start:    now.Add(30 * time.Second),
		End:      now.Add(60 * time.Second),
		Name:     "Some User",
	}

	storage := &apiStorage{reservations: []*Reservation{res}}

	resreq, _ := json.MarshalIndent(res, "", "    ")
	b := bytes.NewBuffer(resreq)

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodPut, "45", b)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("If-Unmodified-Since", now.Format(time.RFC1123))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPutBadJSON(t *testing.T) {
	b := bytes.NewBufferString("this isn't json")
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPut, "44", b)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code 400 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPutNotFound(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:       45,
		Resource: "some resource",
		Start:    now.Add(30 * time.Second),
		End:      now.Add(60 * time.Second),
		Name:     "Some User",
	}

	resreq, _ := json.MarshalIndent(res, "", "    ")
	b := bytes.NewBuffer(resreq)

	handler := v3res(&apiStorage{error: errors.New("not found")})
	r, _ := http.NewRequest(http.MethodPut, "45", b)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 400 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPutError(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:       45,
		Resource: "some resource",
		Start:    now.Add(30 * time.Second),
		End:      now.Add(60 * time.Second),
		Name:     "Some User",
	}

	resreq, _ := json.MarshalIndent(res, "", "    ")
	b := bytes.NewBuffer(resreq)

	handler := v3res(&apiStorage{error: errors.New("oops")})
	r, _ := http.NewRequest(http.MethodPut, "45", b)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code 500 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPutNoRef(t *testing.T) {
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPut, "", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 400 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPutMediaType(t *testing.T) {
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPut, "35", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status code 415 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPutConflict(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:       45,
		Resource: "some resource",
		Start:    now.Add(30 * time.Second),
		End:      now.Add(60 * time.Second),
		Name:     "Some User",
	}

	storage := &apiStorage{
		error:        errors.New("range conflict"),
		reservations: []*Reservation{res},
	}

	resreq, _ := json.MarshalIndent(res, "", "    ")
	b := bytes.NewBuffer(resreq)

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodPut, "35", b)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected status code 409 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPatch(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:       45,
		Resource: "some resource",
		Start:    now.Add(30 * time.Second),
		End:      now.Add(60 * time.Second),
		Name:     "Some User",
		Initials: "SU",
	}

	storage := &apiStorage{reservations: []*Reservation{res}}

	req := fmt.Sprintf(`{"name":"Other User","initials":"OU","loan":true,"end":"%s"}`, time.Now().Add(300*time.Second).Format(time.RFC3339Nano))
	b := bytes.NewBufferString(req)

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodPatch, "45", b)
	r.Header.Set("Content-Type", "application/merge-patch+json")
	r.Header.Set("If-Unmodified-Since", now.Format(time.RFC1123))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 got %s", resp.Status)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPatchModified(t *testing.T) {
	then := time.Now()
	now := time.Now().Add(60 * time.Second)

	res := &Reservation{
		ID:           45,
		LastModified: now,
		Resource:     "some resource",
		Start:        now.Add(30 * time.Second),
		End:          now.Add(60 * time.Second),
		Name:         "Some User",
		Initials:     "SU",
	}

	storage := &apiStorage{reservations: []*Reservation{res}}

	req := fmt.Sprintf(`{"name":"Other User","initials":"OU","loan":true,"end":"%s"}`, time.Now().Add(300*time.Second).Format(time.RFC3339Nano))
	b := bytes.NewBufferString(req)

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodPatch, "45", b)
	r.Header.Set("Content-Type", "application/merge-patch+json")
	r.Header.Set("If-Unmodified-Since", then.Format(time.RFC1123))
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected status code 409 got %s", resp.Status)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPatchWrongType(t *testing.T) {
	now := time.Now()

	res := &Reservation{
		ID:       45,
		Resource: "some resource",
		Start:    now.Add(30 * time.Second),
		End:      now.Add(60 * time.Second),
		Name:     "Some User",
	}

	storage := &apiStorage{reservations: []*Reservation{res}}

	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodPatch, "45", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status code 415 got %s", resp.Status)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPatchNoRef(t *testing.T) {
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPatch, "", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 400 got %d", resp.StatusCode)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPatchNotFound(t *testing.T) {
	storage := &apiStorage{error: errors.New("not found")}
	handler := v3res(storage)
	r, _ := http.NewRequest(http.MethodPatch, "0", nil)
	r.Header.Set("Content-Type", "application/merge-patch+json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 404 got %s", resp.Status)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPatchReadError(t *testing.T) {
	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPatch, "0", &badReader{})
	r.Header.Set("Content-Type", "application/merge-patch+json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code 400 got %s", resp.Status)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestV3APIPatchBadJSON(t *testing.T) {
	b := bytes.NewBufferString("this ain't json")

	handler := v3res(&apiStorage{})
	r, _ := http.NewRequest(http.MethodPatch, "0", b)
	r.Header.Set("Content-Type", "application/merge-patch+json")
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status code 400 got %s", resp.Status)
	}

	exp := "application/json"
	if resp.Header.Get("Content-Type") != exp {
		t.Fatalf("expected content type \"%s\" got \"%s\"", exp, resp.Header.Get("Content-Type"))
	}
}

func TestPatchGeneration(t *testing.T) {
	m := make(map[string]interface{})
	m["name"] = "Some User"
	m["loan"] = true

	b, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(b))
}
