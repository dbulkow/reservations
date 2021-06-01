/* Copyright (c) 2021 David Bulkow */

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	. "github.com/dbulkow/reservations/api"
	"github.com/spf13/cobra"
)

func init() {
	configCmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Write or display configuration fields",
		Long:    "Write or display configuration fields",
		RunE:    config,
	}

	RootCmd.AddCommand(configCmd)
}

type Config struct {
	Name   string `json:"name"`
	Mail   string `json:"mail"`
	Abbrev string `json:"abbrev"`
}

func ConfFile() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home + ".reserve.conf"
	}
	return os.Getenv("HOME") + "/.config/reserve.conf"
}

func getConfig(conffile string) (*Config, error) {
	var cfg = &Config{}

	b, err := ioutil.ReadFile(conffile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	err = json.Unmarshal(b, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func config(cmd *cobra.Command, args []string) error {
	conffile := cmd.Flag("config").Value.String()

	validEmail := func(email string) bool {
		re := regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
		return re.MatchString(email)
	}

	genAbbrev := func(name string) string {
		parts := strings.Split(strings.ToUpper(name), " ")
		var x string
		for _, p := range parts {
			x = x + string(p[0])
		}
		return x
	}

	var cfg Config

	exist := false

	b, err := ioutil.ReadFile(conffile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Unable to read config data %v", err)
	} else if err == nil {
		exist = true
	}

	if exist {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return fmt.Errorf("Unable to read config data %v", err)
		}
	}

	reader := bufio.NewReader(os.Stdin)

	oldname := cfg.Name

	if cfg.Name == "" {
		fmt.Print("Full Name     (First Last): ")
	} else {
		fmt.Printf("Full Name     (First Last, default \"%s\"): ", cfg.Name)
	}
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text != "" {
		cfg.Name = strings.TrimSpace(text)
	}

	if cfg.Name == "" {
		return errors.New("Name not entered")
	}

	oldmail := cfg.Mail

	if cfg.Mail == "" {
		fmt.Print("Email Address: ")
	} else {
		fmt.Printf("Email Address (default \"%s\"): ", cfg.Mail)
	}
	text, _ = reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text != "" {
		cfg.Mail = strings.TrimSpace(text)
	}

	if cfg.Mail == "" {
		return errors.New("Email address not valid")
	}

	if cfg.Abbrev == "" {
		cfg.Abbrev = genAbbrev(cfg.Name)
	}
	fmt.Printf("Abbreviation  ([F]irst [L]ast, default \"%s\"): ", cfg.Abbrev)
	text, _ = reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text != "" {
		cfg.Abbrev = text
	}

	if validEmail(cfg.Mail) == false {
		return errors.New("Email address does not appear to be valid")
	}

	if len(cfg.Abbrev) < 1 || 3 < len(cfg.Abbrev) {
		fmt.Println(cfg.Abbrev)
		return errors.New("Abbreviation needs to be two or three characters")
	}

	cfg.Abbrev = strings.ToUpper(cfg.Abbrev)

	b, err = json.MarshalIndent(&cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("Unable to marshal config data %v", err)
	}

	if err := ioutil.WriteFile(conffile, b, 0666); err != nil {
		return fmt.Errorf("Unable to write config data %v", err)
	}

	if oldname != cfg.Name || oldmail != cfg.Mail {
		err := registerMail(cfg.Name, cfg.Mail)
		if err != nil {
			return err
		}

		fmt.Printf("\nregistered %s\n", cfg.Mail)
	}

	return nil
}

const MaxRead = 1024 * 1024

func registerMail(name, email string) error {
	req := struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}{
		Name:  name,
		Email: email,
	}

	buf, err := json.Marshal(&req)
	if err != nil {
		return err
	}

	b := bytes.NewReader(buf)

	service.Path = V3mail
	r, err := http.NewRequest(http.MethodPost, service.String(), b)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept-Encoding", "gzip")

	client := &http.Client{
		Timeout: time.Duration(10 * time.Second),
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

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("response status %s", resp.Status)
	}

	var reader io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(io.LimitReader(resp.Body, MaxRead))
	default:
		reader = bufio.NewReader(io.LimitReader(resp.Body, MaxRead))
	}

	rpy := struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}{}

	err = json.NewDecoder(reader).Decode(&rpy)
	if err != nil {
		return fmt.Errorf("decode: %v", err)
	}

	if rpy.Status != "Success" {
		return fmt.Errorf("Error: %s", rpy.Error)
	}

	return nil
}
