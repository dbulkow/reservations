/* Copyright (c) 2021 David Bulkow */

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

//go:generate ./version.sh

var RootCmd = &cobra.Command{
	Use:   "reserve",
	Short: "Reserve a resource",
	Long: `Reserve or modify a reservation for a resource

environment:
    RESERVE_URL    URL for reservation server
                   RESERVE_URL_VALUE
    RESERVE_CONFIG config filename
                   RESERVE_CONFIG_VALUE
`,
	PersistentPreRunE: validURL,
}

var (
	service *url.URL
	client  = &http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
)

func validURL(cmd *cobra.Command, args []string) error {
	addr := cmd.Flag("url").Value.String()

	if addr == "" {
		return fmt.Errorf("Error: service URL not set")
	}

	var err error
	service, err = url.Parse(addr)
	if err != nil {
		return fmt.Errorf("Error: service URL invalid %v\n", err)
	}

	return nil
}

func main() {
	var (
		addr   = os.Getenv("RESERVE_URL")
		config = os.Getenv("RESERVE_CONFIG")
	)

	if config == "" {
		config = ConfFile()
	}

	if addr == "" {
		addr = "https://reservations.company.com"
	}

	RootCmd.Long = strings.ReplaceAll(RootCmd.Long, "RESERVE_URL_VALUE", addr)
	RootCmd.Long = strings.ReplaceAll(RootCmd.Long, "RESERVE_CONFIG_VALUE", config)

	RootCmd.PersistentFlags().StringVar(&addr, "url", addr, "URL for reservation service")
	RootCmd.PersistentFlags().StringVar(&config, "config", config, "config file")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display git hash and build data",
		Long:  "Display git hash and build data",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Git Commit Hash: %s\n", GitHash)
			fmt.Printf("Build Time:      %s\n", BuildTime)
		},
	}

	RootCmd.AddCommand(versionCmd)

	err := RootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
}
