/* Copyright (c) 2021 David Bulkow */

package main

import (
	"strings"

	. "github.com/dbulkow/reservations/api"
)

type byID []*Reservation

func (b byID) Len() int      { return len(b) }
func (b byID) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byID) Less(i, j int) bool {
	return b[i].ID < b[j].ID
}

type byDate []*Reservation

func (b byDate) Len() int      { return len(b) }
func (b byDate) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byDate) Less(i, j int) bool {
	return b[i].Start.Before(b[j].Start)
}

type byName []*Reservation

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return strings.Compare(b[i].Name, b[j].Name) < 0 }

type byResource []*Reservation

func (b byResource) Len() int      { return len(b) }
func (b byResource) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byResource) Less(i, j int) bool {
	less := func(i, j int, prefix string) (bool, bool) {
		if strings.HasPrefix(b[i].Resource, prefix) && !strings.HasPrefix(b[j].Resource, prefix) {
			return true, true
		}
		if !strings.HasPrefix(b[i].Resource, prefix) && strings.HasPrefix(b[j].Resource, prefix) {
			return true, false
		}
		if strings.HasPrefix(b[i].Resource, prefix) && strings.HasPrefix(b[j].Resource, prefix) {
			if len(b[i].Resource) > len(prefix) && len(b[j].Resource) > len(prefix) {
				if b[i].Resource[3] > b[j].Resource[3] {
					return true, true
				}
				if b[i].Resource[3] < b[j].Resource[3] {
					return true, false
				}
			}

			cmp := strings.Compare(b[i].Resource, b[j].Resource)

			if cmp == 0 && b[i].Start.Before(b[j].Start) {
				return true, true
			}

			return true, cmp < 0
		}
		return false, false
	}

	if done, retval := less(i, j, "lin"); done {
		return retval
	}

	if done, retval := less(i, j, "esx"); done {
		return retval
	}

	if done, retval := less(i, j, "win"); done {
		return retval
	}

	if done, retval := less(i, j, "poc"); done {
		return retval
	}

	if strings.Compare(b[i].Resource, b[j].Resource) == 0 &&
		b[i].Start.Before(b[j].Start) {
		return true
	}

	return strings.Compare(b[i].Resource, b[j].Resource) < 0
}
