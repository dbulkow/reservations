/* Copyright (c) 2021 David Bulkow */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const RegistrationExpire = time.Duration(24) * time.Hour

// email registration and verification
//
// - user registers with the server
// - email sent to address provided
// - user validates email address by REST GET of provided URL (in email)

type Mail interface {
	Valid(name string) bool
	Lookup(name string) (string, error)
}

type email struct {
	email       string
	uuid        uuid.UUID // unique path for validation
	issued      time.Time // when validation link we issued
	validExpire time.Time // when validation expires
	valid       bool      // user has responded to validate url
}

type mail struct {
	names map[string]*email
	sync.Mutex
}

var MailNameNotFound = errors.New("name not found")

func NewMail(filename string) (*mail, error) {
	m := &mail{
		names: make(map[string]*email),
	}

	// open/read stored data

	return m, nil
}

// look for validated email by name
func (m *mail) Valid(name string) bool {
	m.Lock()
	defer m.Unlock()

	if em, ok := m.names[name]; ok {
		if em.valid {
			return true
		}
	}

	return false
}

// find email by name
// returns error if no email found, or email not validated
func (m *mail) Lookup(name string) (string, error) {
	m.Lock()
	defer m.Unlock()

	if em, ok := m.names[name]; ok {
		if em.valid {
			return em.email, nil
		}
	}

	return "", MailNameNotFound
}

// POST submit name:email
//      returns status
// GET
//      look up by UUID
//      if found, mark email valid
//
// after RegistrationExpire hours, delete email registration (require new registration)

func (m *mail) rest() http.HandlerFunc {
	fail := func(w http.ResponseWriter, errstr string, status int) {
		var resp = struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}{
			Status: "Error",
			Error:  errstr,
		}

		b, err := json.Marshal(&resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(b)
	}

	success := func(w http.ResponseWriter) {
		var resp = struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}{
			Status: "Success",
		}

		b, err := json.Marshal(&resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}

	serve := func(w http.ResponseWriter, filename string) {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write(b)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// extract uuid from path (last element)
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) < 1 {
				fail(w, "invalid path", http.StatusNotFound)
				return
			}

			id, err := uuid.Parse(parts[len(parts)-1])
			if err != nil {
				fail(w, "bad path", http.StatusNotFound)
				return
			}

			m.Lock()
			defer m.Unlock()

			var em *email
			for _, rec := range m.names {
				if rec.uuid == id {
					em = rec
				}
			}

			if em == nil {
				serve(w, "notfound.html")
				return
			}

			if em.valid {
				serve(w, "alreadyvalid.html")
				return
			}

			if time.Now().After(em.validExpire) {
				serve(w, "validexpired.html")
				return
			}

			em.valid = true
			// save mail db to disk
			serve(w, "valid.html")

		case http.MethodPost:
			var req = struct {
				Name  string `json:"name"`
				Email string `json:"request"`
			}{}

			b, err := ioutil.ReadAll(io.LimitReader(r.Body, 65536))
			if err != nil {
				fail(w, "payload read error", http.StatusBadRequest)
				return
			}

			err = json.Unmarshal(b, &req)
			if err != nil {
				fail(w, err.Error(), http.StatusBadRequest)
				return
			}

			m.Lock()
			defer m.Unlock()

			if em, ok := m.names[req.Name]; ok {
				if em.valid {
					fail(w, "name already registered", http.StatusConflict)
					return
				}
			}

			id, err := uuid.NewRandom()
			if err != nil {
				fail(w, "internal error", http.StatusInternalServerError)
				return
			}

			m.names[req.Name] = &email{
				email:       req.Email,
				uuid:        id,
				issued:      time.Now(),
				validExpire: time.Now().Add(RegistrationExpire),
			}

			// delete email registration after it expires
			go func(m *mail, name string) {
				time.Sleep(RegistrationExpire)
				m.Lock()
				defer m.Unlock()
				if em, ok := m.names[name]; ok {
					if em.valid == false {
						delete(m.names, name)
					}
				}
			}(m, req.Name)

			// send email to address provided
			// save mail db to disk

			success(w)

		default:
			http.Error(w, fmt.Sprintf("method \"%s\" not supported", r.Method), http.StatusMethodNotAllowed)
			return
		}
	}
}
