/* Copyright (c) 2021 David Bulkow */

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/dbulkow/reservations/api"
)

var isNumeric = regexp.MustCompile("[0-9]+")

const v3MaxRead = 128 * 1024

func v3res(storage Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "command" {
			v3cmd(storage, w, r)
			return
		}

		var ref int
		var refset bool
		var err error

		if r.URL.Path != "" && r.URL.Path != "*" { // the latter is for OPTIONS
			if !isNumeric.MatchString(r.URL.Path) {
				v3error(w, fmt.Sprintf("ref \"%s\" is not a number", r.URL.Path), http.StatusNotFound)
				return
			}

			ref, err = strconv.Atoi(r.URL.Path)
			if err != nil {
				v3error(w, fmt.Sprintf("ref \"%s\" not a valid number: %v", r.URL.Path, err), http.StatusNotFound)
				return
			}

			refset = true
		}

		switch r.Method {
		case http.MethodOptions:
			if refset {
				w.Header().Set("Allow", "OPTIONS, HEAD, GET, POST, PUT, PATCH, DELETE")
				w.Header().Set("Accept-Patch", "application/json-patch+json, application/merge-patch+json")
			} else {
				w.Header().Set("Allow", "OPTIONS, HEAD, GET, POST")
			}
			w.Header().Set("Content-Length", "0")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			return

		case http.MethodHead:
			fallthrough
		case http.MethodGet:
			if refset {
				v3getref(storage, w, r, ref)
			} else {
				v3get(storage, w, r)
			}

		case http.MethodPost:
			if refset {
				v3error(w, "post not allowed on reservation", http.StatusMethodNotAllowed)
			} else {
				v3post(storage, w, r)
			}

		// the following all require a reference :ref

		case http.MethodPut:
			if refset == false {
				v3error(w, "ref not specified", http.StatusNotFound)
			} else {
				v3put(storage, w, r, ref)
			}

		case http.MethodPatch:
			if refset == false {
				v3error(w, "ref not specified", http.StatusNotFound)
			} else {
				v3patch(storage, w, r, ref)
			}

		case http.MethodDelete:
			if refset == false {
				v3error(w, "ref not specified", http.StatusNotFound)
			} else {
				v3delete(storage, w, r, ref)
			}

		default:
			http.Error(w, fmt.Sprintf("method \"%s\" not supported", r.Method), http.StatusMethodNotAllowed)
		}
	}
}

func v3error(w http.ResponseWriter, errstr string, code int) {
	reply := struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}{}

	reply.Status = "Error"
	reply.Error = errstr

	b, err := json.Marshal(reply)
	if err != nil {
		b = []byte{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(code)
	w.Write(b)
}

func v3getref(storage Storage, w http.ResponseWriter, r *http.Request, ref int) {
	res, err := storage.GetById(ref)
	if err != nil {
		v3error(w, fmt.Sprintf("get %d: %v", ref, err), http.StatusNotFound)
		return
	}

	reply := struct {
		Status      string       `json:"status"`
		Reservation *Reservation `json:"reservation,omitempty"`
	}{
		Status:      "Success",
		Reservation: res,
	}

	b, err := json.Marshal(reply)
	if err != nil {
		v3error(w, fmt.Sprintf("get %d: %v", ref, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Last-Modified", res.LastModified.Format(time.RFC1123))

	since := r.Header.Get("If-Modified-Since")
	t, err := time.Parse(time.RFC1123, since)
	if err == nil {
		fmt.Println(res.LastModified, t)
		if res.LastModified.After(t) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	if r.Method == http.MethodHead {
		return
	}

	w.Write(b)
}

func v3get(storage Storage, w http.ResponseWriter, r *http.Request) {
	var (
		q        = r.URL.Query()
		show     = q.Get("show")
		resource = q.Get("resource")
		count    = q.Get("n")
		last     = q.Get("last")
	)

	start, err := strconv.Atoi(last)
	if err != nil {
		start = 0
	}

	length, err := strconv.Atoi(count)
	if err != nil {
		length = 0
	}

	res, err := storage.List(resource, show, start, length)
	if err != nil {
		v3error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var modified time.Time
	for _, r := range res {
		if r.LastModified.After(modified) {
			modified = r.LastModified
		}
	}

	reply := struct {
		Status       string         `json:"status"`
		Reservations []*Reservation `json:"reservations,omitempty"`
	}{
		Status:       "Success",
		Reservations: res,
	}

	b, err := json.Marshal(reply)
	if err != nil {
		v3error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Last-Modified", modified.Format(time.RFC1123))

	since := r.Header.Get("If-Modified-Since")
	t, err := time.Parse(time.RFC1123, since)
	if err == nil {
		if modified.After(t) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	if r.Method == http.MethodHead {
		return
	}

	w.Write(b)
}

func v3readlen(r *http.Request) int64 {
	clen := r.Header.Get("Content-Length")
	if clen == "" {
		return v3MaxRead
	}

	len, err := strconv.Atoi(clen)
	if err != nil {
		return v3MaxRead
	}

	if len > v3MaxRead {
		return v3MaxRead
	}

	return int64(len)
}

func v3post(storage Storage, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		v3error(w, "request not JSON", http.StatusUnsupportedMediaType)
		return
	}

	reply := struct {
		Status   string `json:"status"`
		Location string `json:"location,omitempty"`
		ID       *int   `json:"id,omitempty"`
	}{}

	var req = &Reservation{}

	err := json.NewDecoder(io.LimitReader(r.Body, v3readlen(r))).Decode(req)
	if err != nil {
		v3error(w, "malformed request", http.StatusBadRequest)
		return
	}

	err = storage.Add(req)
	if err != nil {
		v3error(w, err.Error(), http.StatusBadRequest)
		// StatusConflict
		return
	}

	location := fmt.Sprintf("%s%d", V3path, req.ID)

	reply.Status = "Success"
	reply.Location = location
	reply.ID = &req.ID

	b, err := json.Marshal(reply)
	if err != nil {
		v3error(w, fmt.Sprintf("post: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", location)
	w.Header().Set("ID", strconv.Itoa(req.ID))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Last-Modified", req.LastModified.Format(time.RFC1123))
	w.WriteHeader(http.StatusCreated)
	w.Write(b)
}

// maybe limit this to future reservations?
func v3put(storage Storage, w http.ResponseWriter, r *http.Request, ref int) {
	if r.Header.Get("Content-Type") != "application/json" {
		v3error(w, "request not JSON", http.StatusUnsupportedMediaType)
		return
	}

	var req Reservation

	err := json.NewDecoder(io.LimitReader(r.Body, v3readlen(r))).Decode(&req)
	if err != nil {
		v3error(w, "malformed request", http.StatusBadRequest)
		return
	}

	since := r.Header.Get("If-Unmodified-Since")
	last, err := time.Parse(time.RFC1123, since)
	if err == nil {
		req.LastModified = last
	}

	res, err := storage.Update(ref, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			v3error(w, err.Error(), http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "range conflict") || strings.Contains(err.Error(), "on loan") || strings.Contains(err.Error(), "modified") {
			v3error(w, err.Error(), http.StatusConflict)
			return
		}
		v3error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reply := struct {
		Status      string       `json:"status"`
		Reservation *Reservation `json:"reservation,omitempty"`
	}{
		Status:      "Success",
		Reservation: res,
	}

	b, err := json.Marshal(reply)
	if err != nil {
		v3error(w, fmt.Sprintf("put %d: %v", ref, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Last-Modified", req.LastModified.Format(time.RFC1123))
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func v3patch(storage Storage, w http.ResponseWriter, r *http.Request, ref int) {
	if r.Header.Get("Content-Type") != "application/merge-patch+json" {
		v3error(w, "unknown content type", http.StatusUnsupportedMediaType)
		return
	}

	res, err := storage.GetById(ref)
	if err != nil {
		v3error(w, err.Error(), http.StatusNotFound)
		return
	}

	since := r.Header.Get("If-Unmodified-Since")
	last, err := time.Parse(time.RFC1123, since)
	if err == nil {
		if res.LastModified.After(last) {
			v3error(w, "reservation modified", http.StatusConflict)
			return
		}
	}

	b, err := io.ReadAll(io.LimitReader(r.Body, v3readlen(r)))
	if err != nil {
		v3error(w, "malformed request", http.StatusBadRequest)
		return
	}

	status, err := MergePatch(res, b)
	if err != nil {
		v3error(w, err.Error(), status)
		return
	}

	res, err = storage.Update(res.ID, res)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			v3error(w, err.Error(), http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "range conflict") || strings.Contains(err.Error(), "on loan") || strings.Contains(err.Error(), "modified") {
			v3error(w, err.Error(), http.StatusConflict)
			return
		}
		v3error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reply := struct {
		Status      string       `json:"status"`
		Reservation *Reservation `json:"reservation,omitempty"`
	}{
		Status:      "Success",
		Reservation: res,
	}

	b, err = json.Marshal(reply)
	if err != nil {
		v3error(w, fmt.Sprintf("patch %d: %v", ref, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Last-Modified", res.LastModified.Format(time.RFC1123))
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func v3delete(storage Storage, w http.ResponseWriter, r *http.Request, ref int) {
	err := storage.Delete(ref)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			v3error(w, err.Error(), http.StatusNotFound)
			return
		}
		v3error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func v3cmd(storage Storage, w http.ResponseWriter, r *http.Request) {
	// accept commands in JSON
	// process command
	// return JSON response
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "v3cmd")
}
