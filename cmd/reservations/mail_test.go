package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"testing"
)

func mkmail() *mail {
	return &mail{
		names: map[string]*Email{
			"Some User": &Email{
				Email: "some.user@company.com",
			},
			"Another User": &Email{
				Email: "another.user@company.com",
				Valid: true,
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

	r, _ = http.NewRequest(http.MethodGet, m.names["Third User"].UUID.String(), nil)

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

	if m.names["Third User"].Valid == false {
		t.Fatal("expected valid")
	}
}

func TestMailSaveRestore(t *testing.T) {
	m := mkmail()
	m.filename = "mail_test.json"
	defer os.Remove(m.filename)

	name := struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}{
		Name:  "Third User",
		Email: "third.user@company.com",
	}

	req, _ := json.Marshal(&name)
	b := bytes.NewBuffer(req)
	handler := m.rest()
	r, _ := http.NewRequest(http.MethodPost, "", b)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	if false {
		data, err := ioutil.ReadFile(m.filename)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(string(data))
	}

	m.names = make(map[string]*Email)

	err := m.readfile()
	if err != nil {
		t.Fatal(err)
	}

	exp := "third.user@company.com"
	if m.names["Third User"].Email != exp {
		t.Fatalf("expected \"%s\" got \"%s\"", exp, m.names["Third User"].Email)
	}
}
