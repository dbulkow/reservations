/* Copyright (c) 2021 David Bulkow */

package main

import (
	"errors"
	"log"
	"sync"
	"time"

	. "github.com/dbulkow/reservations/api"
)

type BackingStore interface {
	Add(*Reservation) error
	Update(int, *Reservation) error
	Delete(int) error
	ReadLog(*memory) error
}

type memory struct {
	nextID       int
	reservations []*Reservation
	store        BackingStore
	mail         Mail
	sync.Mutex
}

type nonstore struct{}

func (s *nonstore) Add(*Reservation) error         { return nil }
func (s *nonstore) Update(int, *Reservation) error { return nil }
func (s *nonstore) Delete(int) error               { return nil }
func (s *nonstore) ReadLog(*memory) error          { return nil }

func NewMemory(store BackingStore, mail Mail) (*memory, error) {
	m := &memory{
		reservations: make([]*Reservation, 0),
		mail:         mail,
	}

	if store == nil {
		m.store = &nonstore{}
	} else {
		m.store = store
	}

	err := store.ReadLog(m)
	if err != nil {
		return nil, err
	}

	return m, nil
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
			// string is empty on error, which is what we want
			res.Email, _ = m.mail.Lookup(res.Name)
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

		if start > 0 && res.ID < start {
			continue
		}

		if length > 0 && len(response) >= length {
			continue
		}

		switch show {
		case "current": // active reservations
			// in the future or in the past and not on loan
			if now.Before(res.Start) || (now.After(res.End) && res.Loan == false) {
				continue
			}

		case "history": // expired reservations
			if now.Before(res.End) || res.Loan {
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

		// string is empty on error, which is what we want
		res.Email, _ = m.mail.Lookup(res.Name)

		response = append(response, res)
	}

	return response, nil
}

// add new reservation - no overlaps allowed
func (m *memory) Add(res *Reservation) error {
	m.Lock()
	defer m.Unlock()

	// let's not be so restrictive - maybe limit unregistered user to short reservations (no loans)
	// if m.mail.Valid(res.Name) == false {
	// 	return errors.New("unknown name")
	// }

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
	res.Email = ""
	res.LastModified = time.Now().Round(time.Second)

	if res.Loan {
		res.End = res.Start
	}

	m.nextID++
	m.reservations = append(m.reservations, res)

	err := m.store.Add(res)
	if err != nil {
		return err
	}

	log.Printf("added %s", res)

	return nil
}

// replace reservation if no overlap
// don't allow:
// - update of start or end if active or expired
// - update of loan if active
// - update of ID
// - update if res.LastModified newer than req.LastModified
func (m *memory) Update(ref int, req *Reservation) (*Reservation, error) {
	res, err := m.GetById(ref)
	if err != nil {
		return nil, err
	}

	if res.LastModified.After(req.LastModified) {
		return nil, errors.New("modified")
	}

	m.Lock()
	defer m.Unlock()

	now := time.Now()

	if res.End.Before(now) && res.Loan == false {
		return nil, errors.New("already expired")
	}

	// if active - only allow notes, share and end time changes
	if res.Start.Before(now) {
		if req.Resource != res.Resource || req.Start != res.Start {
			return nil, errors.New("already active")
		}

		if res.Loan != req.Loan {
			return nil, errors.New("converting to/from loan")
		}

		res.LastModified = now.Round(time.Second)
		res.End = req.End
		res.Notes = req.Notes
		res.Share = req.Share
		res.Name = req.Name
		res.Initials = req.Initials
		res.Email = ""

		err := m.store.Update(res.ID, res)
		if err != nil {
			return nil, err
		}

		log.Printf("updated %s", res)

		return res, nil
	}

	res.LastModified = now.Round(time.Second)
	res.Resource = req.Resource
	res.Start = req.Start
	res.End = req.End
	res.Loan = req.Loan
	res.Share = req.Share
	res.Notes = req.Notes
	res.Name = req.Name
	res.Initials = req.Initials
	res.Email = ""

	err = m.store.Update(res.ID, res)
	if err != nil {
		return nil, err
	}

	log.Printf("updated %s", res)

	return res, nil
}

// if reservation start is in the future, just delete it
// if reservation end is in the past, ignore this request
// if reservation is active (start < now and (end > now || on loan))
//    remove loan flag
//    set end time <= now
func (m *memory) Delete(ref int, lastmod time.Time) error {
	m.Lock()
	defer m.Unlock()

	now := time.Now()

	for i, r := range m.reservations {
		if r.ID != ref {
			continue
		}

		if r.LastModified.After(lastmod) {
			return errors.New("resource modified")
		}

		if r.Start.After(now) {
			m.reservations = append(m.reservations[:i], m.reservations[i+1:]...)

			err := m.store.Delete(ref)
			if err != nil {
				return err
			}

			log.Println("deleted", ref)

			return nil
		}

		if r.Loan {
			r.Loan = false
			r.End = now
			r.LastModified = time.Now().Round(time.Second)

			err := m.store.Update(r.ID, r)
			if err != nil {
				return err
			}

			log.Println("ended", ref)

			return nil
		}

		if r.Start.Before(now) && r.End.After(now) {
			r.End = now
			r.LastModified = time.Now().Round(time.Second)

			err := m.store.Update(r.ID, r)
			if err != nil {
				return err
			}

			log.Println("ended", ref)

			return nil
		}

		if r.End.Before(now) {
			return errors.New("resource already expired")
		}
	}

	return errors.New("resource not found")
}
