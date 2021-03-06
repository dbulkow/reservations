/* Copyright (c) 2021 David Bulkow */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"strings"

	. "github.com/dbulkow/reservations/api"
	"github.com/spf13/cobra"
)

var (
	long       bool
	quiet      bool
	jsonOutput bool
	current    bool
	sortby     string
	showres    bool
	history    bool
	showall    bool
	mine       bool
	numres     int
)

func init() {
	listCmd := &cobra.Command{
		Use:     "list [<resource name or prefix>]",
		Aliases: []string{"ls"},
		Short:   "List reservations",
		Long: `List reservations

When run with no arguments this command lists reservations for all
resources. Specific resources, or resources sharing a prefix, can be
listed by specifying these on the command line.

Flags can be added to limit results to one's own reservations, set the
sort order, list the history of a resource and more.
`,
		RunE: list,
	}

	listCmd.Flags().BoolVarP(&long, "long", "l", false, "Long listing")
	listCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Don't display header")
	listCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "JSON output")
	listCmd.Flags().StringVar(&sortby, "sort-by", "resource", "Sort by [date, resource, name, id]")
	listCmd.Flags().BoolVarP(&showres, "showres", "r", false, "Show reservation number")
	listCmd.Flags().BoolVar(&history, "history", false, "Include reservation history")
	listCmd.Flags().BoolVar(&showall, "all", false, "Show all reservations, history, current, future")
	listCmd.Flags().BoolVarP(&mine, "mine", "m", false, "Show your reservations only")
	listCmd.Flags().BoolVarP(&current, "current", "c", false, "List active reservations")
	listCmd.Flags().IntVarP(&numres, "num", "n", 50, "Number of reservations to retrieve each request")

	RootCmd.AddCommand(listCmd)
}

// fix date display
//  - today, just time 3pm
//  - tomorrow 3pm
//  - within a week, day name 3pm
//  - day name, date, time

func list(cmd *cobra.Command, args []string) error {
	conffile := cmd.Flag("config").Value.String()
	cfg, err := getConfig(conffile)
	if err != nil {
		return fmt.Errorf("Unable to read config (%v).  Run with 'config' to initialize.", err)
	}

	service.Path = V3api

	u, err := url.Parse(service.String())
	if err != nil {
		return err
	}
	q := u.Query()

	if current {
		q.Set("show", "current")
	} else if history {
		q.Set("show", "history")
	} else if showall {
		q.Set("show", "all")
	}

	q.Set("limit", strconv.Itoa(numres))
	q.Set("start", "0")
	u.RawQuery = q.Encode()

	var res []*Reservation

	for {
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
		}{}

		err = json.NewDecoder(io.LimitReader(resp.Body, MaxRead)).Decode(&rpy)
		if err != nil {
			return fmt.Errorf("decode: %v", err)
		}

		if rpy.Status != "Success" {
			return errors.New(rpy.Error)
		}

		if rpy.Reservations == nil {
			break
		}

		for _, r := range rpy.Reservations {
			res = append(res, r)
		}

		next := resp.Header.Get("X-Next-Reservation")
		if next == "" {
			break
		}

		u, err = url.Parse(next)
		if err != nil {
			return err
		}
	}

	var filter string
	if len(args) > 0 {
		filter = args[0]
	}

	datefmt := "Jan _2 15:04 2006"

	var (
		reslen   = len("ID")
		machlen  = len("Resource")
		namelen  = len("Name")
		datelen  = len(datefmt)
		hasDates = false
		hasShare = false
	)

	if !long && !jsonOutput {
		for _, r := range res {
			if !strings.HasPrefix(r.Resource, filter) {
				continue
			}
			if mine && filter == "" && r.Name != cfg.Name {
				continue
			}
			id := fmt.Sprintf("%d", r.ID)
			if len(id) > reslen {
				reslen = len(id)
			}
			if len(r.Resource) > machlen {
				machlen = len(r.Resource)
			}
			if len(r.Name) > namelen {
				namelen = len(r.Name)
			}
			if !r.Loan {
				hasDates = true
			}
			if r.Share {
				hasShare = true
			}
		}
	}

	switch sortby {
	case "resource":
		sort.Sort(byResource(res))
	case "name":
		sort.Sort(byName(res))
	case "date":
		sort.Sort(byDate(res))
	case "id":
		sort.Sort(byID(res))
	}

	if !quiet && !jsonOutput {
		if long {
			fmt.Println("reservation          details")
			fmt.Println("-----------          -------")
		} else {
			if showres {
				fmt.Printf("%-*s ", reslen, "ID")
			}
			fmt.Printf("%-*s ", machlen, "Resource")
			if hasShare {
				fmt.Printf("%-5s ", "Share")
			}
			fmt.Printf("%-*s ", namelen, "Name")
			if hasDates {
				fmt.Printf("%-*s   %-*s\n", datelen, "Start", datelen, "End")
			} else {
				fmt.Println(" Loan")
			}
			if showres {
				fmt.Printf("%-*s ", reslen, strings.Repeat("-", reslen))
			}
			fmt.Printf("%-*s ", machlen, strings.Repeat("-", machlen))
			if hasShare {
				fmt.Printf("%-5s ", "-----")
			}
			fmt.Printf("%-*s", namelen, strings.Repeat("-", namelen))
			if hasDates {
				fmt.Printf(" %-*s   %-*s\n", datelen, strings.Repeat("-", datelen), datelen, strings.Repeat("-", datelen))
			} else {
				fmt.Printf(" %s\n", strings.Repeat("-", len("On Loan")))
			}
		}
	}

	if jsonOutput {
		fmt.Print("[")
	}

	var lastResource string
	for _, r := range res {
		if !strings.HasPrefix(r.Resource, filter) {
			continue
		}
		if mine && filter == "" && r.Name != cfg.Name {
			continue
		}
		start := r.Start.Local().Format(datefmt)
		end := r.End.Local().Format(datefmt)
		if long {
			canshare := ""
			if r.Share {
				canshare = " (can share)"
			}
			fmt.Printf("%5d\t   Resource: %s%s\n", r.ID, r.Resource, canshare)
			if r.Loan {
				fmt.Printf("\tReservation: On Loan\n")
			} else {
				fmt.Printf("\tReservation: %s - %s\n", start, end)
			}
			fmt.Printf("\t       Name: %s", r.Name)
			if r.Email == "" {
				fmt.Printf("\n")
			} else {
				fmt.Printf("(%s)\n", r.Email)
			}
			if r.Notes != "" {
				fmt.Printf("\t      Notes: %s\n", r.Notes)
			}
			fmt.Println()
		} else if jsonOutput {
			b, err := json.Marshal(&r)
			if err != nil {
				return fmt.Errorf("unable to marshal output %v", err)
			}

			fmt.Println(string(b))
		} else {
			canshare := "     "
			if r.Share {
				canshare = " yes "
			}
			if showres {
				fmt.Printf("%-*d ", reslen, r.ID)
			}
			resource := r.Resource
			if resource == lastResource {
				resource = ""
			}
			lastResource = r.Resource
			fmt.Printf("%-*s ", machlen, resource)
			if hasShare {
				fmt.Printf("%-5s ", canshare)
			}
			fmt.Printf("%-*s ", namelen, r.Name)
			if r.Loan {
				fmt.Printf("On Loan\n")
			} else {
				// adjust start/end to more human readable values
				fmt.Printf("%-*s - %-*s\n", datelen, start, datelen, end)
			}
		}
	}

	if jsonOutput {
		fmt.Println("]")
	}

	return nil
}
