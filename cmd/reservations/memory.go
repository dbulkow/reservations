/* Copyright (c) 2021 David Bulkow */

package main

import (
	"errors"
	"sync"
	"time"

	. "github.com/dbulkow/reservations/api"
)

type memory struct {
	nextID       int
	reservations []*Reservation
	mail         Mail
	sync.Mutex
}

func NewMemory( /*backing store*/ ) (*memory, error) {
	// load from backing store
	return &memory{reservations: make([]*Reservation, 0), mail: &mail{}}, nil
}

// determine if the two reservation time ranges overlap each other
func (m *memory) overlap(s, r *Reservation) bool {
	return s.Start.Before(r.End) && s.End.After(r.Start)
}

// read array from end because active entries will be closer to end
func (m *memory) GetById(resid int) (*Reservation, error) {
	m.Lock()
	defer m.Unlock()

	for i := len(m.reservations) - 1; i >= 0; i-- {
		res := m.reservations[i]
		if res.ID == resid {
			return res, nil
		}
	}

	return nil, errors.New("reservation not found")
}

func (m *memory) List(resource, show string, start, length int) ([]*Reservation, error) {
	m.Lock()
	defer m.Unlock()

	response := make([]*Reservation, 0)

	now := time.Now()

	for _, res := range m.reservations {
		if resource != "" && res.Resource != resource {
			continue
		}

		switch show {
		case "current": // active reservations
			if now.Before(res.Start) || now.After(res.End) {
				continue
			}
			if now.After(res.Start) && (now.Before(res.End) || res.Loan) {
			}

		case "history": // expired reservations
			if now.Before(res.End) {
				continue
			}

		case "all": // everything

		case "active": // active and future reservations
			fallthrough
		default:
			if now.After(res.End) && res.Loan == false {
				continue
			}
		}

		response = append(response, res)
	}

	return response, nil
}

// add new reservation - no overlaps allowed
func (m *memory) Add(res *Reservation) error {
	m.Lock()
	defer m.Unlock()

	if m.mail.Valid(res.Name) == false {
		return errors.New("unknown name")
	}

	for _, r := range m.reservations {
		if r.Resource != res.Resource {
			continue
		}

		if r.Loan {
			return errors.New("resource on loan")
		}

		if m.overlap(r, res) {
			return errors.New("reservation range conflict")
		}
	}

	res.ID = m.nextID

	if res.Loan {
		res.End = res.Start
	}

	m.nextID++
	m.reservations = append(m.reservations, res)
	return nil
}

// replace reservation if no overlap
// etag in res must match the reservation in ref
// don't allow:
// - update of start or end if before now
// - update of loan
// - update of ID
// - update if ref.LastModified newer than res.LastModified
func (m *memory) Update(ref int, res *Reservation) (*Reservation, error) {
	m.Lock()
	defer m.Unlock()

	return nil, nil
}

// if reservation start is in the future, just delete it
// if reservation end is in the past, ignore this request
// if reservation is active (start < now and (end > now || on loan))
//    remove loan flag
//    set end time <= now
func (m *memory) Delete(ref int) error {
	m.Lock()
	defer m.Unlock()

	now := time.Now()

	for i, r := range m.reservations {
		if r.ID != ref {
			continue
		}

		if r.Start.After(now) {
			m.reservations = append(m.reservations[:i], m.reservations[i+1:]...)
			return nil
		}

		if r.Loan {
			r.Loan = false
			r.End = now
			return nil
		}

		if r.Start.Before(now) && r.End.After(now) {
			r.End = now
			return nil
		}

		if r.End.Before(now) {
			return errors.New("resource already expired")
		}
	}

	return errors.New("resource not found")
}
