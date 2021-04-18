/* Copyright (c) 2021 David Bulkow */

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
)

const (
	log_responses = false
	log_requests  = false
)

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			log.Printf("[%s] Path => %s User agent => %s Remote addr => %s", r.Method, r.URL.Path, r.UserAgent(), r.RemoteAddr)
		}

		if log_requests {
			dump, err := httputil.DumpRequest(r, true)
			if err != nil {
				http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
				return
			}

			log.Printf("%s", dump)
		}

		if log_responses {
			rec := httptest.NewRecorder()

			next.ServeHTTP(rec, r)

			b, _ := httputil.DumpResponse(rec.Result(), false)

			if rec.HeaderMap.Get("Content-Type") == "application/json" {
				log.Println(string(b))
			}

			// this copies the recorded response to the response writer
			for k, v := range rec.HeaderMap {
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			rec.Body.WriteTo(w)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
