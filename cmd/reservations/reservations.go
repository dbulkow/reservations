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
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dbulkow/reservations/internal/getenv"
	. "github.com/dbulkow/reservations/internal/gzip"
)

// favicon from from http://clipartbarn.com/clock-clip-art_36285/
//go:embed favicon.ico
var assets embed.FS

func run(args []string, stdout, stderr io.Writer) error {
	var (
		env = getenv.NewEnv("RESERVATIONS")

		port = env.Get("PORT", "8080")
		addr = env.Get("ADDR", "localhost")
	)

	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	flags.StringVar(&port, "port", port, "REST/HTTP port number")
	flags.StringVar(&addr, "addr", addr, "Listen address")

	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s\n", args[0])
		fmt.Fprintln(stderr)
		fmt.Fprintf(stderr, `Environment variables:
  RESERVATIONS_PORT = %s
        HTTP listen port
  RESERVATIONS_ADDR = %s
        Network listen address
`, port, addr)
		flags.PrintDefaults()
	}

	err := flags.Parse(args[1:])
	if err != nil {
		return err
	}

	// report version details

	log.Printf("git commit hash: %s\n", "xxx")
	log.Printf("build time:      %s\n", "xxx")

	// server initialization

	ctxt, cancel := context.WithCancel(context.Background())
	defer cancel()

	var jobs sync.WaitGroup

	// filename := fmt.Sprintf("%s-%s", prefix, time.Now().Format("20060102"))
	file, err := NewJSONL("reservations.jsonl")
	if err != nil {
		return err
	}

	storage, err := NewMemory(file)
	if err != nil {
		return err
	}

	// XXX load from backing store

	// http routes

	mux := http.NewServeMux()
	mux.Handle("/", Gzip(logger(http.FileServer(http.FS(assets)))))
	mux.Handle("/help", Gzip(logger(http.HandlerFunc(usage))))
	mux.Handle(v3path, Gzip(logger(http.StripPrefix("/v3/reservations/", http.HandlerFunc(v3res(storage))))))

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
