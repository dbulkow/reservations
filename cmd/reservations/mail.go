/* Copyright (c) 2021 David Bulkow */

package main

// mail registration and verification

type Mail interface {
	Valid(name string) bool
}

type mail struct{}

// look for registered and validated email
func (m *mail) Valid(name string) bool {
	return true
}
