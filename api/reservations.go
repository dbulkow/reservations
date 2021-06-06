/* Copyright (c) 2021 David Bulkow */

package api

import (
	"fmt"
	"time"
)

type Reservation struct {
	ID           int       `json:"id"`
	LastModified time.Time `json:"lastModified"`
	Resource     string    `json:"resource"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Loan         bool      `json:"loan"`
	Share        bool      `json:"share"`
	Notes        string    `json:"notes,omitempty"`
	Name         string    `json:"name"`
	Initials     string    `json:"initials"`
	Email        string    `json:"email"`
}

const (
	V3mail = "/v3/mailverify"
	V3api  = "/v3/reservations/"
)

func (r *Reservation) String() string {
	if r.Loan {
		return fmt.Sprintf("%d %s loan %s", r.ID, r.Resource, r.Name)
	} else {
		return fmt.Sprintf("%d %s from %s to %s %s", r.ID, r.Resource, r.Start.Format(time.RFC3339), r.End.Format(time.RFC3339), r.Name)
	}
}
