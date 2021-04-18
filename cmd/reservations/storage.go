/* Copyright (c) 2021 David Bulkow */

package main

import . "github.com/dbulkow/reservations/api"

type Storage interface {
	Get() (*Reservation, error)
	List(show string, start, length int) ([]*Reservation, error)
	Add(res *Reservation) error
	Update(ref int, res *Reservation) error
	Delete(ref int) error
}
