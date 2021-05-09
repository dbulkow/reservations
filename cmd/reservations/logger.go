/* Copyright (c) 2021 David Bulkow */

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
)

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			log.Printf("[%s] Path => %s User agent => %s Remote addr => %s", r.Method, r.URL.Path, r.UserAgent(), r.RemoteAddr)
		}

		request, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}

		response := httptest.NewRecorder()
		next.ServeHTTP(response, r)

		if response.Code >= http.StatusBadRequest {
			log.Println(string(request))

			content := response.HeaderMap.Get("Content-Type")
			body := content == "application/json"

			out, _ := httputil.DumpResponse(response.Result(), body)
			log.Println(string(out))
		}

		// this copies the recorded response to the response writer
		for k, v := range response.HeaderMap {
			w.Header()[k] = v
		}
		w.WriteHeader(response.Code)
		response.Body.WriteTo(w)
	})
}
