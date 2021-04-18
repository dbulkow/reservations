/* Copyright (c) 2021 David Bulkow */

package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	//	. "github.com/dbulkow/reservations/api"
)

var isNumeric = regexp.MustCompile("[0-9]+")

func v3res(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "command" {
		v3cmd(w, r)
		return
	}

	var ref int
	var refset bool
	var err error

	if r.URL.Path != "" {
		if !isNumeric.MatchString(r.URL.Path) {
			http.Error(w, fmt.Sprintf("ref \"%s\" is not a number", r.URL.Path), http.StatusNotFound)
			return
		}

		ref, err = strconv.Atoi(r.URL.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("ref \"%s\" is not a number: %v", r.URL.Path, err), http.StatusNotFound)
		}

		refset = true
	}

	switch r.Method {
	case http.MethodGet:
		if refset {
			v3getref(w, r, ref)
			return
		}

		v3get(w, r)

	case http.MethodPost:
		if refset {
			http.Error(w, "post not allowed on reservation", http.StatusMethodNotAllowed)
			return
		}
		v3post(w, r)

	// the following all require a reference :ref

	case http.MethodPut:
		if refset == false {
			http.Error(w, "ref not specified", http.StatusNotFound)
			return
		}
		v3put(w, r, ref)

	case http.MethodPatch:
		if refset == false {
			http.Error(w, "ref not specified", http.StatusNotFound)
			return
		}
		v3patch(w, r, ref)

	case http.MethodDelete:
		if refset == false {
			http.Error(w, "ref not specified", http.StatusNotFound)
			return
		}
		v3delete(w, r, ref)

	default:
		http.Error(w, fmt.Sprintf("method \"%s\" not supported", r.Method), http.StatusMethodNotAllowed)
	}
}

func v3getref(w http.ResponseWriter, r *http.Request, ref int) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "get %d", ref)
}

func v3get(w http.ResponseWriter, r *http.Request) {
	var (
		q        = r.URL.Query()
		show     = q.Get("show")
		resource = q.Get("resource")
		num      = q.Get("n")
		last     = q.Get("last")
		rel      = q.Get("rel")
	)

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "get\n")
	fmt.Fprintf(w, "   show=%s\n", show)
	fmt.Fprintf(w, "   resource=%s\n", resource)
	fmt.Fprintf(w, "   n=%s\n", num)
	fmt.Fprintf(w, "   last=%s\n", last)
	fmt.Fprintf(w, "   rel=%s\n", rel)
}

func v3post(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "post")
}

func v3put(w http.ResponseWriter, r *http.Request, ref int) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "put %d", ref)
}

func v3patch(w http.ResponseWriter, r *http.Request, ref int) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "patch %d", ref)
}

func v3delete(w http.ResponseWriter, r *http.Request, ref int) {
	// look up by reference
	// if not found return
	// if active return
	// if history return
	// delete reference
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "delete %d", ref)
}

func v3cmd(w http.ResponseWriter, r *http.Request) {
	// accept commands in JSON
	// process command
	// return JSON response
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "v3cmd")
}
