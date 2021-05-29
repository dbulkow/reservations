/* Copyright (c) 2021 David Bulkow */

package api

import "time"

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
