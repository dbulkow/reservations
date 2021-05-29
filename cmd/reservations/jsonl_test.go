package main

import (
	"os"
	"testing"
	"time"

	. "github.com/dbulkow/reservations/api"
)

func TestJSONL(t *testing.T) {
	filename := time.Now().Format("reservations-20060102150405000000.jsonl")

	// fmt.Println(filename)

	js, err := NewJSONL(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filename)

	res := &Reservation{
		ID:       56,
		Resource: "resource",
	}

	err = js.Add(res)
	if err != nil {
		t.Fatal(err)
	}

	res.Start = time.Now()

	err = js.Update(res.ID, res)
	if err != nil {
		t.Fatal(err)
	}

	err = js.Delete(res.ID)
	if err != nil {
		t.Fatal(err)
	}

	// b, err := ioutil.ReadFile(filename)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	//
	// fmt.Println(string(b))

	m := &memory{
		reservations: make([]*Reservation, 0),
	}

	err = js.ReadLog(m)
	if err != nil {
		t.Fatal(err)
	}
}
