/* Copyright (c) 2021 David Bulkow */

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	. "github.com/dbulkow/reservations/api"
	"github.com/spf13/cobra"
)

var (
	canshare bool
	notes    string
	onloan   bool
	dryrun   bool
)

func init() {
	addCmd := &cobra.Command{
		Use:     "add <resource> <time specification>",
		Aliases: []string{"new", "create"},
		Short:   "Add a Reservation",
		Long: `Add a reservation, accepting a limited time specification to provide
start and end times.  The start time can be implicit:

    reserve add <resource> <duration>

Durations can be specified in hours, days or weeks as:

    + 5 days
    plus 7 hours
    for 1 week

The start and end times can be explicit, separated by to or until:

    reserve add <resource> from <start> to <end>

or with a duration:

    Friday 8am for 3 days
    April 1st 8am plus 7 hours

where the time specification for start/end can be:

    9pm
    05:00pm
    21:00
    2017-04-01 08:00
    October 15th 09:00 [optional year] (note: does not support 9am)
    tomorrow 8am
    Thursday noon

Synonyms for times are:

    noon
    midnight
    eod -> 5pm
`,
		RunE: add,
	}

	addCmd.Flags().BoolVar(&canshare, "share", false, "Can share")
	addCmd.Flags().StringVar(&notes, "notes", "", "Notes")
	addCmd.Flags().BoolVar(&onloan, "loan", false, "On Loan")
	addCmd.Flags().BoolVarP(&dryrun, "dryrun", "n", false, "Just print out parsed time")

	RootCmd.AddCommand(addCmd)
}

func add(cmd *cobra.Command, args []string) error {
	conffile := cmd.Flag("config").Value.String()
	cfg, err := getConfig(conffile)
	if err != nil {
		return fmt.Errorf("Unable to read config (%v).  Run with 'config' to initialize.", err)
	}

	service.Path = V3api

	if onloan {
		if len(args) < 1 {
			return errors.New("resource not specified")
		}
	} else {
		if len(args) < 2 {
			return errors.New("resource and/or duration not specified")
		}
	}

	resource := args[0]
	start := time.Now()
	end := time.Now()

	if !onloan {
		start, end, err = ParseRange(time.Now(), args[1:])
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

		if dryrun {
			fmt.Println(start, end)
			return nil
		}
	}

	res := &Reservation{
		Resource: resource,
		Start:    start,
		End:      end,
		Loan:     onloan,
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

	r, err := http.NewRequest(http.MethodPost, service.String(), b)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")

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

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("response status %s", resp.Status)
	}

	rpy := struct {
		Status   string `json:"status"`
		Error    string `json:"error"`
		Location string `json:"location"`
		ID       *int   `json:"id"`
	}{}

	err = json.NewDecoder(io.LimitReader(resp.Body, MaxRead)).Decode(&rpy)
	if err != nil {
		return fmt.Errorf("decode %v", err)
	}

	if rpy.Status != "Success" {
		return fmt.Errorf("error: %s", rpy.Error)
	}

	if rpy.ID != nil {
		fmt.Printf("Added reservation %d\n", *rpy.ID)
	}

	return nil
}
