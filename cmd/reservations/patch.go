/* Copyright (c) 2021 David Bulkow */

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	. "github.com/dbulkow/reservations/api"
)

func MergePatch(res *Reservation, patch []byte) (int, error) {
	var p interface{}

	err := json.Unmarshal(patch, &p)
	if err != nil {
		return http.StatusBadRequest, err
	}

	m := p.(map[string]interface{})

	for k, v := range m {
		switch vv := v.(type) {
		case string:
			// fmt.Println(k, "is string", vv)

			switch k {
			case "resource":
				res.Resource = vv
			case "start":
				t, err := time.Parse(time.RFC3339Nano, vv)
				if err != nil {
					return http.StatusBadRequest, errors.New("time field malformed")
				}
				res.Start = t
			case "end":
				t, err := time.Parse(time.RFC3339Nano, vv)
				if err != nil {
					return http.StatusBadRequest, errors.New("time field malformed")
				}
				res.End = t
			case "name":
				res.Name = vv
			case "initials":
				res.Initials = vv
			case "notes":
				res.Notes = vv
			default:
				return http.StatusBadRequest, errors.New("unknown field name")
			}

		case bool:
			// fmt.Println(k, "is bool", vv)

			switch k {
			case "loan":
				res.Loan = vv
			case "share":
				res.Share = vv
			default:
				return http.StatusBadRequest, errors.New("unknown field name")
			}
		default:
			return http.StatusBadRequest, errors.New("unknown field type")
		}
	}

	return http.StatusOK, nil
}
