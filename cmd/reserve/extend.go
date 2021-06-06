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
	"os"
	"strings"
	"time"

	. "github.com/dbulkow/reservations/api"
	"github.com/spf13/cobra"
)

func init() {
	extendCmd := &cobra.Command{
		Use:   "extend <resource> <time specification>",
		Short: "Extend an active reservation",
		Long: `Extend an active reservation by the duration or specified end time

See add command for details of time specification
`,
		RunE: extend,
	}

	extendCmd.Flags().BoolVar(&canshare, "share", false, "Can share")
	extendCmd.Flags().StringVar(&notes, "notes", "", "Notes")

	RootCmd.AddCommand(extendCmd)
}

func extend(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("resource and/or duration not specified")
	}

	// read the current reservation for a resource

	service.Path = V3api

	u, err := url.Parse(service.String())
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("show", "current")
	q.Set("resource", args[0])
	u.RawQuery = q.Encode()

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
		out, err := httputil.DumpResponse(resp, false)
		if err != nil {
			log.Println(err)
		}

		fmt.Println(string(out))
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response status: %s", resp.Status)
	}

	rpy := struct {
		Status       string         `json:"status"`
		Error        string         `json:"error"`
		Reservations []*Reservation `json:"reservations"`
		Reservation  *Reservation   `json:"reservation"`
	}{}

	err = json.NewDecoder(io.LimitReader(resp.Body, MaxRead)).Decode(&rpy)
	if err != nil {
		return fmt.Errorf("decode: %v", err)
	}

	if rpy.Status != "Success" {
		return errors.New(rpy.Error)
	}

	if rpy.Reservations == nil {
		return errors.New("empty reservation in response")
	}

	res := rpy.Reservations[0]
	end := res.End.In(time.Local)

	end, err = ParseDuration(end, args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsetime: %v\n", err)
		if perr, ok := err.(*ParseError); ok {
			if perr.token == nil {
				goto done
			}
			tokens, _ := tokenize(args[1:])
			for i, t := range tokens.tokens {
				if perr.token.count == i+1 {
					fmt.Printf("[%s] ", t.Val)
				} else {
					fmt.Printf("%s ", t.Val)
				}
			}
			fmt.Println()
		}
	done:
		os.Exit(1)
	}

	// send a Patch request with updated fields

	var patch strings.Builder
	fmt.Fprintf(&patch, `{"end":"%s"`, end.Format(time.RFC3339Nano))
	if notes != "" {
		fmt.Fprintf(&patch, `, "notes":"%s"`, notes)
	}
	if canshare != res.Share {
		fmt.Fprintf(&patch, `, "share":%t`, canshare)
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
