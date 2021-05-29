package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
)

func mkmail() *mail {
	return &mail{
		names: map[string]*email{
			"Some User": &email{
				email: "some.user@company.com",
			},
			"Another User": &email{
				email: "another.user@company.com",
				valid: true,
			},
		},
	}
}

func TestMailValid(t *testing.T) {
	m := mkmail()

	if m.Valid("Some User") != false {
		t.Fatal("some user not valid")
	}

	if m.Valid("Another User") != true {
		t.Fatal("another user not valid")
	}
}

func TestMailLookup(t *testing.T) {
	m := mkmail()

	_, err := m.Lookup("Some User")
	if err != MailNameNotFound {
		t.Fatal("some user found")
	}

	_, err = m.Lookup("Another User")
	if err != nil {
		t.Fatalf("another user %v", err)
	}
}

func TestMailRest(t *testing.T) {
	name := struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}{
		Name:  "Third User",
		Email: "third.user@company.com",
	}

	req, _ := json.Marshal(&name)
	b := bytes.NewBuffer(req)

	m := mkmail()
	handler := m.rest()
	r, _ := http.NewRequest(http.MethodPost, "", b)
	r.Header.Set("Content-Type", "application/json")

	in, err := httputil.DumpRequest(r, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(in))

	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()

	out, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	// for _, em := range m.names {
	// 	fmt.Printf("%+v\n", em)
	// }

	r, _ = http.NewRequest(http.MethodGet, m.names["Third User"].uuid.String(), nil)

	in, err = httputil.DumpRequest(r, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(in))

	w = httptest.NewRecorder()
	handler(w, r)

	resp = w.Result()

	out, err = httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(out))

	// for _, em := range m.names {
	// 	fmt.Printf("%+v\n", em)
	// }

	if m.names["Third User"].valid == false {
		t.Fatal("expected valid")
	}
}
