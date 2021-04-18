/* Copyright (c) 2021 David Bulkow */

package main

import (
	"fmt"
	"net/http"
	"regexp"
)

const usetext = `Reservations Server

GET    /v3/reservations/         - get all reservations
GET    /v3/reservations/<index>  - get one reservation
POST   /v3/reservations/         - create reservation
PUT    /v3/reservations/<index>  - update reservation
PATCH  /v3/reservations/<index>  - update reservation
DELETE /v3/reservations/<index>  - delete reservation
`

var browserAgents = regexp.MustCompile("Mozilla|AppleWebKit|WebKit|Chrome|Safari")

func usage(w http.ResponseWriter, r *http.Request) {
	if !browserAgents.MatchString(r.UserAgent()) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, usetext)
		return
	}

	// respond with fancy version
}
