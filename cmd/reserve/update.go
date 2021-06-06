/* Copyright (c) 2021 David Bulkow */

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	. "github.com/dbulkow/reservations/api"
	"github.com/spf13/cobra"
)

func init() {
	updateCmd := &cobra.Command{
		Use:   "update <resource id number>",
		Short: "Update a reservation notes and/or share flag",
		Long:  "Update a reservation notes and/or share flag",
		RunE:  update,
	}

	updateCmd.Flags().BoolVar(&canshare, "share", false, "Can share")
	updateCmd.Flags().StringVar(&notes, "notes", "", "Notes")

	RootCmd.AddCommand(updateCmd)
}

func update(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource id not specified")
	}

	resid, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	// read the current reservation for the resid

	service.Path = V3api

	u, err := url.Parse(fmt.Sprintf("%s%d", service, resid))
	if err != nil {
		return err
	}

	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}

	if false {
		in, err := httputil.DumpRequest(r, false)
		if err != nil {
			log.Println(err)
		}

		fmt.Println(string(in))
	}

	resp, err := client.Do(r)
	if err != nil {
		return fmt.Errorf("http: %v", err)
	}
	if resp == nil {
		return fmt.Errorf("empty response")
	}
	defer func() {
		io.Copy(ioutil.Discard, io.LimitReader(resp.Body, MaxRead))
		resp.Body.Close()
	}()

	if false {
		out, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.Println(err)
		}

		fmt.Println(string(out))
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response status: %s", resp.Status)
	}

	rpy := struct {
		Status      string       `json:"status"`
		Error       string       `json:"error"`
		Reservation *Reservation `json:"reservation"`
	}{}

	err = json.NewDecoder(io.LimitReader(resp.Body, MaxRead)).Decode(&rpy)
	if err != nil {
		return fmt.Errorf("decode: %v", err)
	}

	if rpy.Status != "Success" {
		return errors.New(rpy.Error)
	}

	if rpy.Reservation == nil {
		return errors.New("empty reservation in response")
	}

	res := rpy.Reservation

	// send a Patch request

	var patch strings.Builder
	var comma bool
	fmt.Fprintf(&patch, `{`)
	if notes != "" {
		if comma {
			fmt.Fprintf(&patch, `, `)
		}
		fmt.Fprintf(&patch, `"notes":"%s"`, notes)
		comma = true
	}
	if canshare != res.Share {
		if comma {
			fmt.Fprintf(&patch, `, `)
		}
		fmt.Fprintf(&patch, `"share":%t`, canshare)
		comma = true
	}
	fmt.Fprintf(&patch, `}`)

	b := bytes.NewBufferString(patch.String())

	u, err = url.Parse(fmt.Sprintf("%s%d", service.String(), res.ID))
	if err != nil {
		return err
	}

	r, err = http.NewRequest(http.MethodPatch, u.String(), b)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}
	r.Header.Set("Content-Type", "application/merge-patch+json")
	r.Header.Set("If-Unmodified-Since", resp.Header.Get("Last-Modified"))

	if false {
		in, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Println(err)
		}

		fmt.Println(string(in))
	}

	resp, err = client.Do(r)
	if err != nil {
		return fmt.Errorf("http: %v", err)
	}
	if resp == nil {
		return fmt.Errorf("empty response")
	}
	defer func() {
		io.Copy(ioutil.Discard, io.LimitReader(resp.Body, MaxRead))
		resp.Body.Close()
	}()

	if false {
		out, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.Println(err)
		}

		fmt.Println(string(out))
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response status %s", resp.Status)
	}

	err = json.NewDecoder(io.LimitReader(resp.Body, MaxRead)).Decode(&rpy)
	if err != nil {
		return fmt.Errorf("decode %v", err)
	}

	if rpy.Status != "Success" {
		return fmt.Errorf("error: %s", rpy.Error)
	}

	if rpy.Reservation == nil {
		return errors.New("empty reservation in response")
	}

	res = rpy.Reservation

	fmt.Printf("updated reservation %d\n", res.ID)

	return nil
}
