/* Copyright (c) 2021 David Bulkow */

package api

import "time"

type Reservation struct {
	ID       int
	Resource string
	Start    time.Time
	End      time.Time
	Loan     bool
	Share    bool
	Name     string
	Email    string
	Initials string
	Notes    string
}
