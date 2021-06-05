/* Copyright (c) 2021 David Bulkow */

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	. "github.com/dbulkow/reservations/api"
	"github.com/spf13/cobra"
)

func init() {
	deleteCmd := &cobra.Command{
		Use:     "delete <resevation id number>",
		Aliases: []string{"rm", "del"},
		Short:   "Delete a future reservation",
		Long: `Delete a future reservation

Use the end command to terminate a loan or active reservation.
Expired reservations can not be deleted as they represent a history
of resource utilization.
`,
		RunE: delete,
	}

	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "Force remove, don't prompt")

	RootCmd.AddCommand(deleteCmd)
}

var force bool

func delete(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("reservation id not specified")
	}

	resid, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	service.Path = fmt.Sprintf("%s%d", V3api, resid)

	r, err := http.NewRequest(http.MethodGet, service.String(), nil)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response status %s", resp.Status)
	}

	rpy := struct {
		Status      string       `json:"status"`
		Error       string       `json:"error"`
		Reservation *Reservation `json:"reservation"`
	}{}

	err = json.NewDecoder(io.LimitReader(resp.Body, MaxRead)).Decode(&rpy)
	if err != nil {
		return fmt.Errorf("decode %v", err)
	}

	if rpy.Status != "Success" {
		return fmt.Errorf("error: %s", rpy.Error)
	}

	if rpy.Reservation == nil {
		return fmt.Errorf("reservation %d missing data", resid)
	}

	res := rpy.Reservation

	datefmt := "Jan _2 15:04 2006"
	fmt.Println("Delete the following entry:")
	fmt.Printf("\n%d %s %s %s %s\n", res.ID, res.Resource, res.Name, res.Start.Local().Format(datefmt), res.End.Local().Format(datefmt))

	if force == false {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nProceed? (y/N) ")
		text, _ := reader.ReadString('\n')

		if strings.ToLower(strings.TrimSpace(text)) != "y" {
			return errors.New("cancelled")
		}
	}

	r, err = http.NewRequest(http.MethodDelete, service.String(), nil)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
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
