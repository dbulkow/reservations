/* Copyright (c) 2021 David Bulkow */

package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/dbulkow/reservations/api"
)

type memtestMailer struct {
	valid bool
}

func (m *memtestMailer) Valid(string) bool             { return m.valid }
func (m *memtestMailer) Lookup(string) (string, error) { return "", nil }

func fillMemory(valid bool) (*memory, time.Time) {
	storage := &memory{store: &nonstore{}}

	now := time.Now()

	storage.mail = &memtestMailer{valid: valid}
	storage.nextID = 120
	storage.reservations = []*Reservation{
		&Reservation{
			ID:           35,
			LastModified: now,
			Resource:     "resource A",
			Start:        now.Add(30 * time.Second),
			End:          now.Add(60 * time.Second),
		},
		&Reservation{
			ID:           78,
			LastModified: now,
			Resource:     "resource A",
			Start:        now.Add(30 * time.Hour),
			End:          now.Add(60 * time.Hour),
		},
		&Reservation{
			ID:           79,
			LastModified: now,
			Resource:     "resource B",
			Start:        now.Add(60 * time.Second),
			End:          now.Add(180 * time.Second),
		},
		&Reservation{
			ID:           80,
			LastModified: now,
			Resource:     "resource C",
			Start:        now.Add(90 * time.Second),
			End:          now.Add(100 * time.Second),
		},
		&Reservation{
			ID:           110,
			LastModified: now,
			Resource:     "resource C",
			Start:        now.Add(100 * time.Second),
			End:          now.Add(120 * time.Second),
		},
		&Reservation{
			ID:           111,
			LastModified: now,
			Resource:     "resource D",
			Start:        now.Add(90 * time.Second),
			End:          now.Add(100 * time.Second),
		},
		&Reservation{
			ID:           112,
			LastModified: now,
			Resource:     "resource X",
			Start:        now,
			End:          now,
			Loan:         true,
		},
		&Reservation{
			ID:           113,
			LastModified: now,
			Resource:     "resource Y",
			Start:        now,
			End:          now.Add(10 * time.Second),
		},
		&Reservation{
			ID:           114,
			LastModified: now,
			Resource:     "resource Z",
			Start:        now,
			End:          now,
		},
	}

	return storage, now
}

func TestMemoryGetById(t *testing.T) {
	storage, _ := fillMemory(true)

	id := 110

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	if res.ID != id {
		fmt.Printf("%+v\n", res)
		t.Fatalf("expected res id \"%d\", got \"%d\"", id, res.ID)
	}
}

func TestMemoryGetByIdNotFound(t *testing.T) {
	storage, _ := fillMemory(true)

	_, err := storage.GetById(120)
	if err == nil {
		t.Fatalf("expected \"not found\" error")
	}

	if strings.Contains(err.Error(), "not found") == false {
		t.Fatalf("expected \"not found\" error, got \"%s\"", err.Error())
	}
}

func TestMemoryList(t *testing.T) {
	storage, _ := fillMemory(true)

	count := len(storage.reservations)

	res, err := storage.List("", "all", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != count {
		t.Fatalf("expected %d reservations got %d", count, len(res))
	}

	res, err = storage.List("resource A", "all", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 2 {
		t.Fatalf("expected %d reservations got %d", 2, len(res))
	}

	time.Sleep(50 * time.Millisecond)

	res, err = storage.List("", "current", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 2 {
		t.Fatalf("expected %d reservations got %d", 2, len(res))
	}

	res, err = storage.List("", "history", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 1 {
		t.Fatalf("expected %d reservations got %d", 1, len(res))
	}

	res, err = storage.List("", "all", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != len(storage.reservations) {
		t.Fatalf("expected %d reservations got %d", len(storage.reservations), len(res))
	}

	res, err = storage.List("", "active", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 8 {
		t.Fatalf("expected %d reservations got %d", 8, len(res))
	}
}

func TestMemoryAdd(t *testing.T) {
	storage, now := fillMemory(true)

	res := &Reservation{
		Resource: "resource D",
		Start:    now.Add(100 * time.Second),
		End:      now.Add(120 * time.Second),
	}

	err := storage.Add(res)
	if err != nil {
		t.Fatal(err)
	}

	if storage.nextID != 121 {
		t.Fatalf("expected next ID \"%d\", got \"%d\"", 121, storage.nextID)
	}
}

func TestMemoryAddOverlap(t *testing.T) {
	storage, now := fillMemory(true)

	res := &Reservation{
		Resource: "resource D",
		Start:    now.Add(95 * time.Second),
		End:      now.Add(120 * time.Second),
	}

	err := storage.Add(res)
	if err == nil {
		t.Fatal("expected conflict error")
	}

	if strings.Contains(err.Error(), "range conflict") == false {
		t.Fatalf("expected an error with \"range conflict\" got \"%s\"", err.Error())
	}

	if storage.nextID != 120 {
		t.Fatalf("expected next ID \"%d\", got \"%d\"", 120, storage.nextID)
	}

	res.Start = now.Add(80 * time.Second)
	res.End = now.Add(95 * time.Second)

	err = storage.Add(res)
	if err == nil {
		t.Fatal("expected conflict error")
	}

	if strings.Contains(err.Error(), "range conflict") == false {
		t.Fatalf("expected an error with \"range conflict\" got \"%s\"", err.Error())
	}

	if storage.nextID != 120 {
		t.Fatalf("expected next ID \"%d\", got \"%d\"", 120, storage.nextID)
	}
}

func TestMemoryAddExistingLoan(t *testing.T) {
	storage, now := fillMemory(true)

	resloan := &Reservation{
		Resource: "resource E",
		Start:    now.Add(100 * time.Second),
		End:      now.Add(120 * time.Second),
		Loan:     true,
	}

	err := storage.Add(resloan)
	if err != nil {
		t.Fatal(err)
	}

	res := &Reservation{
		Resource: "resource E",
		Start:    now.Add(100 * time.Second),
		End:      now.Add(120 * time.Second),
		Loan:     true,
	}

	err = storage.Add(res)
	if err == nil {
		t.Fatal("expected \"on loan\" error")
	}

	if strings.Contains(err.Error(), "on loan") == false {
		t.Fatalf("expected an error with \"on loan\" got \"%s\"", err.Error())
	}
}

/*
func TestMemoryAddUnknownName(t *testing.T) {
	storage, now := fillMemory(false)

	res := &Reservation{
		Resource: "resource D",
		Start:    now.Add(100 * time.Second),
		End:      now.Add(120 * time.Second),
		Name:     "Frank Mistfowler",
	}

	err := storage.Add(res)
	if err == nil {
		t.Fatal("expected error \"unknown name\"")
	}

	if strings.Contains(err.Error(), "unknown name") == false {
		t.Fatalf("expected an error with \"unknown name\" got \"%s\"", err.Error())
	}
}
*/
func TestMemoryUpdate(t *testing.T) {
	storage, now := fillMemory(true)

	id := 35

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	req := &Reservation{
		LastModified: res.LastModified,
		Resource:     res.Resource,
		Start:        res.Start,
		End:          now.Add(1 * time.Hour),
		Loan:         res.Loan,
		Share:        res.Share,
		Notes:        res.Notes,
		Name:         "test person",
		Initials:     res.Initials,
	}

	res, err = storage.Update(id, req)
	if err != nil {
		t.Fatal(err)
	}

	if res.Name != req.Name {
		t.Fatalf("name not modified")
	}

	if res.End != req.End {
		t.Fatalf("end time not modified")
	}
}

func TestMemoryUpdateActive(t *testing.T) {
	storage, now := fillMemory(true)

	time.Sleep(50 * time.Millisecond)

	id := 113

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	req := &Reservation{
		LastModified: res.LastModified,
		Resource:     res.Resource,
		Start:        res.Start,
		End:          now.Add(1 * time.Hour),
		Loan:         res.Loan,
		Share:        res.Share,
		Notes:        res.Notes,
		Name:         "test person",
		Initials:     res.Initials,
	}

	res, err = storage.Update(id, req)
	if err != nil {
		t.Fatal(err)
	}

	if res.Name != req.Name {
		t.Fatalf("name not modified")
	}

	if res.End != req.End {
		t.Fatalf("end time not modified")
	}
}

func TestMemoryUpdateModified(t *testing.T) {
	storage, now := fillMemory(true)

	id := 35

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	res.LastModified = res.LastModified.Add(time.Second)

	req := &Reservation{
		LastModified: now,
		Resource:     res.Resource,
		Start:        res.Start,
		End:          now.Add(1 * time.Hour),
		Loan:         res.Loan,
		Share:        res.Share,
		Notes:        res.Notes,
		Name:         "test person",
		Initials:     res.Initials,
	}

	res, err = storage.Update(id, req)
	if err == nil {
		t.Fatal("expected \"not modifed\" error")
	}

	if strings.Contains(err.Error(), "modified") == false {
		t.Fatalf("expected \"modified\" got \"%s\"", err.Error())
	}
}

func TestMemoryUpdateExpired(t *testing.T) {
	storage, now := fillMemory(true)

	time.Sleep(50 * time.Millisecond)

	id := 114

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	req := &Reservation{
		LastModified: res.LastModified,
		Resource:     res.Resource,
		Start:        res.Start,
		End:          now.Add(1 * time.Hour),
		Loan:         res.Loan,
		Share:        res.Share,
		Notes:        res.Notes,
		Name:         "test person",
		Initials:     res.Initials,
	}

	res, err = storage.Update(id, req)
	if err == nil {
		t.Fatal("expected \"already expired\" error")
	}

	if strings.Contains(err.Error(), "already expired") == false {
		t.Fatalf("expected \"already expired\" got \"%s\"", err.Error())
	}
}

func TestMemoryUpdateAlreadyActive(t *testing.T) {
	storage, now := fillMemory(true)

	time.Sleep(50 * time.Millisecond)

	id := 113

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	req := &Reservation{
		LastModified: res.LastModified,
		Resource:     "new resource",
		Start:        res.Start,
		End:          now.Add(1 * time.Hour),
		Loan:         res.Loan,
		Share:        res.Share,
		Notes:        res.Notes,
		Name:         "test person",
		Initials:     res.Initials,
	}

	res, err = storage.Update(id, req)
	if err == nil {
		t.Fatal("expected \"already active\" error")
	}

	if strings.Contains(err.Error(), "already active") == false {
		t.Fatalf("expected \"already active\" got \"%s\"", err.Error())
	}
}

func TestMemoryUpdateConvertActiveLoan(t *testing.T) {
	storage, now := fillMemory(true)

	time.Sleep(50 * time.Millisecond)

	id := 113

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	req := &Reservation{
		LastModified: res.LastModified,
		Resource:     res.Resource,
		Start:        res.Start,
		End:          now.Add(1 * time.Hour),
		Loan:         !res.Loan,
		Share:        res.Share,
		Notes:        res.Notes,
		Name:         "test person",
		Initials:     res.Initials,
	}

	res, err = storage.Update(id, req)
	if err == nil {
		t.Fatal("expected \"converting\" error")
	}

	if strings.Contains(err.Error(), "converting to/from loan") == false {
		t.Fatalf("expected \"converting to/from loan\" got \"%s\"", err.Error())
	}
}

func TestMemoryDelete(t *testing.T) {
	storage, _ := fillMemory(true)

	id := 78

	err := storage.Delete(id)
	if err != nil {
		t.Fatal(err)
	}

	_, err = storage.GetById(id)
	if err == nil {
		t.Fatal("expected \"not found\" error")
	}
}

func TestMemoryDeleteLoan(t *testing.T) {
	storage, _ := fillMemory(true)

	id := 112

	err := storage.Delete(id)
	if err != nil {
		t.Fatal(err)
	}

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	if res.Loan {
		t.Fatalf("expected loan false got %t", res.Loan)
	}

	if res.End.After(res.Start) == false {
		t.Fatalf("expected end time after start")
	}
}

func TestMemoryDeleteActive(t *testing.T) {
	storage, _ := fillMemory(true)

	id := 113

	res, err := storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	end := res.End

	err = storage.Delete(id)
	if err != nil {
		t.Fatal(err)
	}

	res, err = storage.GetById(id)
	if err != nil {
		t.Fatal(err)
	}

	if end.After(res.End) == false {
		t.Fatal("end time not adjusted")
	}
}

func TestMemoryDeleteExpired(t *testing.T) {
	storage, _ := fillMemory(true)

	id := 114

	err := storage.Delete(id)
	if err == nil {
		t.Fatal("expected \"already expired\" error")
	}

	if strings.Contains(err.Error(), "already expired") == false {
		t.Fatalf("expected \"already expired\" error, got \"%s\"", err.Error())
	}
}

func TestMemoryDeleteNotFound(t *testing.T) {
	storage, _ := fillMemory(true)

	id := 1000

	err := storage.Delete(id)
	if err == nil {
		t.Fatal("expected \"not found\" error")
	}

	if strings.Contains(err.Error(), "not found") == false {
		t.Fatalf("expected \"not found\" error, got \"%s\"", err.Error())
	}
}
