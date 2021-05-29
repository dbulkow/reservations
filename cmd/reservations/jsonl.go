/* Copyright (c) 2021 David Bulkow */

//
// Saves a log of reservation updates in JSONL format.
// JSONL is one line per record, each record in JSON. It is
// not an array.
//

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	. "github.com/dbulkow/reservations/api"
)

type jsonl struct {
	file     *os.File
	filename string
}

func NewJSONL(filename string) (*jsonl, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return &jsonl{filename: filename}, nil
}

type jsonlog struct {
	Operation   string       `json:"op"`
	ID          int          `json:"id"`
	Reservation *Reservation `json:"res"`
}

func (j *jsonl) Add(res *Reservation) error {
	var record jsonlog

	record.Operation = "add"
	record.ID = res.ID
	record.Reservation = res

	return j.append(&record)
}

func (j *jsonl) Update(ref int, res *Reservation) error {
	var record jsonlog

	record.Operation = "modify"
	record.ID = ref
	record.Reservation = res

	return j.append(&record)
}

func (j *jsonl) Delete(ref int) error {
	var record jsonlog

	record.Operation = "delete"
	record.ID = ref

	return j.append(&record)
}

func (j *jsonl) append(record *jsonlog) error {
	file, err := os.OpenFile(j.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(record)
	if err != nil {
		return fmt.Errorf("jsonl encode: %v", err)
	}

	return nil
}

func (j *jsonl) ReadLog(m *memory) error {
	file, err := os.Open(j.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record jsonlog

		err := json.Unmarshal(scanner.Bytes(), &record)
		if err != nil {
			return err
		}

		switch record.Operation {
		case "add":
			m.reservations = append(m.reservations, record.Reservation)
		case "modify":
			for i, r := range m.reservations {
				if r.ID != record.ID {
					continue
				}

				m.reservations[i] = record.Reservation
			}
		case "delete":
			for i, r := range m.reservations {
				if r.ID != record.ID {
					continue
				}

				m.reservations = append(m.reservations[:i], m.reservations[i+1:]...)
				break
			}
		default:
			return fmt.Errorf("unknown log operation: %s", record.Operation)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
