/* Copyright (c) 2021 David Bulkow */

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
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

type Email struct {
	Email  string    `json:"email"`
	UUID   uuid.UUID `json:"uuid"`   // unique path for validation
	Expire time.Time `json:"expire"` // when validation expires
	Valid  bool      `json:"valid"`  // user has responded to validate url
}

type mail struct {
	names    map[string]*Email
	filename string
	server   string // mail server address
	port     string // mail server port
	from     string // sender email address
	sync.Mutex
}

var MailNameNotFound = errors.New("name not found")

func NewMail(filename, server, port, from string) (*mail, error) {
	m := &mail{
		names:    make(map[string]*Email),
		filename: filename,
		server:   server,
		port:     port,
		from:     from,
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	file.Close()

	err = m.readfile()
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	}

	return m, nil
}

func (m *mail) readfile() error {
	if m.filename == "" {
		return nil
	}

	file, err := os.Open(m.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(&m.names)
}

func (m *mail) savefile() error {
	if m.filename == "" {
		return nil
	}

	newfile := m.filename + "-"

	file, err := os.Create(newfile)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ")
	err = enc.Encode(&m.names)
	if err != nil {
		return err
	}

	err = os.Rename(newfile, m.filename)
	if err != nil {
		return err
	}

	return nil
}

// look for validated email by name
func (m *mail) Valid(name string) bool {
	m.Lock()
	defer m.Unlock()

	if em, ok := m.names[name]; ok {
		if em.Valid {
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
		if em.Valid {
			return em.Email, nil
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
		}{
			Status: "Success",
		}

		b, err := json.Marshal(&resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
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

			var email *Email
			var name string
			for n, em := range m.names {
				if em.UUID == id {
					email = em
					name = n
				}
			}

			if email == nil {
				log.Printf("email for id %s not found", id.String())
				serve(w, "notfound.html")
				return
			}

			if email.Valid {
				log.Printf("email %s already valid", email.Email)
				serve(w, "alreadyvalid.html")
				return
			}

			if time.Now().After(email.Expire) {
				log.Printf("email %s validation expired", email.Email)
				serve(w, "validexpired.html")
				return
			}

			email.Valid = true
			log.Printf("email verified (%s=%s)", name, email.Email)

			err = m.savefile()
			if err != nil {
				log.Printf("mail post: %v", err)
			}

			serve(w, "valid.html")

		case http.MethodPost:
			var req = struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{}

			var reader io.Reader
			var err error

			switch r.Header.Get("Content-Encoding") {
			case "gzip":
				reader, err = gzip.NewReader(io.LimitReader(r.Body, 65536))
			default:
				reader = bufio.NewReader(io.LimitReader(r.Body, 65536))
			}

			b, err := ioutil.ReadAll(reader)
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
				if em.Valid {
					fail(w, "name already registered", http.StatusConflict)
					return
				}
			}

			for _, em := range m.names {
				if em.Email == req.Email {
					fail(w, "email already registered", http.StatusConflict)
					return
				}
			}

			id, err := uuid.NewRandom()
			if err != nil {
				fail(w, "internal error", http.StatusInternalServerError)
				return
			}

			m.names[req.Name] = &Email{
				Email:  req.Email,
				UUID:   id,
				Expire: time.Now().Add(RegistrationExpire),
			}

			m.sendmail(req.Email, id.String())

			err = m.savefile()
			if err != nil {
				// log.Printf("mail post: %v", err)
			}

			success(w)

		case http.MethodPut:
			// allow email updates?
			// how to close the loop - send an email to old address first?
			// send email to new address, delete old one after verified?
			// need to avoid users changing email for others
			// would like this to remain self-service

		default:
			http.Error(w, fmt.Sprintf("method \"%s\" not supported", r.Method), http.StatusMethodNotAllowed)
			return
		}
	}
}

func (m *mail) sendmail(target, uuid string) error {
	if m.server == "" {
		return nil
	}

	body := fmt.Sprintf(`To: %s\r
Subject: Please verify your email address\r
\r
Someone has registered this email address for use in the reservation\r
service. If you received this mail in error, please ignore and your\r
email will not be used.\r
\r
Please vist the following URL to verify:\r
\r
    https://reservations.company.com/mail/%s\r
`, target, uuid)

	c, err := smtp.Dial(net.JoinHostPort(m.server, m.port))
	if err != nil {
		return err
	}
	defer c.Close()

	err = c.Mail(m.from)
	if err != nil {
		return err
	}

	err = c.Rcpt(target)
	if err != nil {
		// log.Printf("unable to add \"%s\" as recipient: %v", r, err)
	}

	// Send the mail body
	wc, err := c.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(wc, body)
	if err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	// Sends the QUIT command and close the connection
	return c.Quit()
}
