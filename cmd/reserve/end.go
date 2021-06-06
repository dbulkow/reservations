/* Copyright (c) 2021 David Bulkow */

package main

import (
	"bufio"
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

	. "github.com/dbulkow/reservations/api"
	"github.com/spf13/cobra"
)

func init() {
	endCmd := &cobra.Command{
		Use:     "end <resource>",
		Short:   "End an active reservation or loan immediately",
		Long:    "End an active reservation or loan immediately",
		Aliases: []string{"release", "surrender", "giveup"},
		RunE:    end,
	}

	endCmd.Flags().BoolVarP(&force, "force", "f", false, "Force remove, don't prompt")

	RootCmd.AddCommand(endCmd)
}

func end(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("resource name not specified")
	}

	// grab the current reservation for a resource

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

	if false {
		out, err := httputil.DumpResponse(resp, false)
		if err != nil {
			log.Println(err)
		}

		fmt.Println(string(out))
	}

	rpy := struct {
		Status       string         `json:"status"`
		Error        string         `json:"error"`
		Reservations []*Reservation `json:"reservations"`
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

	// if we found a reservation, delete it

	if len(rpy.Reservations) < 1 {
		return errors.New("no matching reservations")
	}

	res := rpy.Reservations[0]

	datefmt := "Jan _2 15:04 2006"
	fmt.Println("End the following reservation:")
	if res.Loan {
		fmt.Printf("\n%d %s %s loan\n", res.ID, res.Resource, res.Name)
	} else {
		fmt.Printf("\n%d %s %s %s %s\n", res.ID, res.Resource, res.Name, res.Start.Local().Format(datefmt), res.End.Local().Format(datefmt))
	}

	if force == false {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nProceed? (y/N) ")
		text, _ := reader.ReadString('\n')

		if strings.ToLower(strings.TrimSpace(text)) != "y" {
			return errors.New("cancelled")
		}
	}

	u, err = url.Parse(fmt.Sprintf("%s%d", service.String(), res.ID))
	if err != nil {
		return err
	}

	r, err = http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}
	r.Header.Set("If-Unmodified-Since", resp.Header.Get("Last-Modified"))

	if false {
		in, err := httputil.DumpRequest(r, false)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response status %s", resp.Status)
	}

	return nil
}
