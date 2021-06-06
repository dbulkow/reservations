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

	. "github.com/dbulkow/reservations/api"
	"github.com/spf13/cobra"
)

func init() {
	nextCmd := &cobra.Command{
		Use:   "next <resource> <time specification>",
		Short: "Reserve a resource next, after current reservation if one exists",
		Long: `Reserve a resource next, after current reservation if one exists

See add command for details of time specification
`,
		RunE: next,
	}

	nextCmd.Flags().BoolVar(&canshare, "share", false, "Can share")
	nextCmd.Flags().StringVar(&notes, "notes", "", "Notes")

	RootCmd.AddCommand(nextCmd)
}

func next(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("resource and/or duration not specified")
	}

	conffile := cmd.Flag("config").Value.String()
	cfg, err := getConfig(conffile)
	if err != nil {
		return fmt.Errorf("Unable to read config (%v).  Run with 'config' to initialize.", err)
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

	start := res.End
	end, err := ParseDuration(start, args[1:])
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

	// add a new reservation

	res = &Reservation{
		Resource: args[0],
		Start:    start,
		End:      end,
		Share:    canshare,
		Notes:    notes,
		Name:     cfg.Name,
		Initials: cfg.Abbrev,
	}

	data, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("marshal %v", err)
	}

	b := bytes.NewReader(data)

	u, err = url.Parse(service.String())
	if err != nil {
		return err
	}

	r, err = http.NewRequest(http.MethodPost, u.String(), b)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")

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

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("response status %s", resp.Status)
	}

	addrpy := struct {
		Status   string `json:"status"`
		Error    string `json:"error"`
		Location string `json:"location"`
		ID       *int   `json:"id"`
	}{}

	err = json.NewDecoder(io.LimitReader(resp.Body, MaxRead)).Decode(&addrpy)
	if err != nil {
		return fmt.Errorf("decode %v", err)
	}

	if addrpy.Status != "Success" {
		return fmt.Errorf("error: %s", addrpy.Error)
	}

	if addrpy.ID == nil {
		return errors.New("empty reply")
	}

	fmt.Printf("Added reservation %d\n", *addrpy.ID)

	return nil
}
