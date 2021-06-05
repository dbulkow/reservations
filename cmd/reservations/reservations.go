/* Copyright (c) 2021 David Bulkow */

package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	. "github.com/dbulkow/reservations/api"
	"github.com/dbulkow/reservations/internal/getenv"
	_ "github.com/dbulkow/reservations/internal/gzip"
)

// favicon from from http://clipartbarn.com/clock-clip-art_36285/
//go:embed favicon.ico
var assets embed.FS

//go:generate ./version.sh

var service *url.URL

func run(args []string, stdout, stderr io.Writer) error {
	var (
		env = getenv.NewEnv("RESERVATIONS")

		port = env.Get("PORT", "8080")
		addr = env.Get("ADDR", "localhost")

		datafile = env.Get("DATA", "reservations.jsonl")
		mailfile = env.Get("MAIL", "mail.json")
	)

	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	flags.StringVar(&port, "port", port, "REST/HTTP port number")
	flags.StringVar(&addr, "addr", addr, "Listen address")
	flags.StringVar(&datafile, "data", datafile, "Backing store filename")
	flags.StringVar(&mailfile, "mail", mailfile, "Mail registration filename")

	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s\n", args[0])
		fmt.Fprintln(stderr)
		fmt.Fprintf(stderr, `Environment variables:
  RESERVATIONS_PORT = %s
        HTTP listen port
  RESERVATIONS_ADDR = %s
        Network listen address
  RESERVATIONS_DATA = %s
        Backing store filename
  RESERVATIONS_MAIL = %s
        Mail registrations filename
`, port, addr, datafile, mailfile)
		flags.PrintDefaults()
	}

	err := flags.Parse(args[1:])
	if err != nil {
		return err
	}

	// report version details

	log.Printf("git commit hash: %s\n", GitHash)
	log.Printf("build time:      %s\n", BuildTime)

	// server initialization

	ctxt, cancel := context.WithCancel(context.Background())
	defer cancel()

	var jobs sync.WaitGroup

	// filename := fmt.Sprintf("%s-%s", prefix, time.Now().Format("20060102"))
	file, err := NewJSONL(datafile)
	if err != nil {
		return err
	}

	mail, err := NewMail(mailfile, "" /*server*/, "" /*port*/, "" /*from*/)
	if err != nil {
		return err
	}

	storage, err := NewMemory(file, mail)
	if err != nil {
		return err
	}

	// XXX load from backing store

	// http routes

	mux := http.NewServeMux()
	mux.Handle("/", logger(http.FileServer(http.FS(assets))))
	mux.Handle("/help", logger(http.HandlerFunc(usage)))
	mux.Handle(V3api, logger(http.StripPrefix(V3api, http.HandlerFunc(v3res(storage)))))
	mux.Handle(V3mail, logger(mail.rest()))
	mux.Handle(V3mail+"/", logger(mail.rest()))

	srv := &http.Server{
		Addr:           net.JoinHostPort(addr, port),
		Handler:        mux,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSNextProto:   nil,
	}

	// signal handling

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-c

		log.Println("signal received")
		log.Println("stopping web server")

		err := srv.Shutdown(ctxt)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("stopping background tasks")

		cancel()
	}()

	// start web listener

	// the service is convenient for development but will not
	// reflect reality through a load balancer or proxy
	service, err = url.Parse(fmt.Sprintf("http://%s:%s", addr, port))
	if err != nil {
		return err
	}

	log.Printf("serving http at %s", net.JoinHostPort(addr, port))

	err = srv.ListenAndServe()
	if err != nil {
		log.Println(err)
	}

	// graceful exit

	log.Println("http server stopped")
	log.Println("waiting for active jobs")

	jobs.Wait()

	log.Println("exiting")

	return nil
}

func main() {
	err := run(os.Args, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
