package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/jmanero/static/pkg/static"
	"github.com/spf13/pflag"
)

// Flag Registers
var (
	ListenFlag string
	HelpFlag   bool
)

// Logger for main routine
var Logger = log.New(os.Stderr, "", log.LUTC|log.Ldate|log.Ltime)

func init() {
	pflag.StringVar(&ListenFlag, "listen", "127.0.0.1:9807", "Set the service's listener address/port")
	pflag.BoolVar(&HelpFlag, "help", false, "Print usage and exit")

	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Serve static files from a directory tree on disk")
		fmt.Fprintf(os.Stderr, "  %s [flags] DIRECTORY\n", os.Args[0])
		pflag.PrintDefaults()
	}
}

func main() {
	pflag.Parse()

	if HelpFlag || pflag.NArg() != 1 {
		pflag.Usage()
		os.Exit(1)
	}

	root, err := filepath.Abs(pflag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to resolve root path", err)
		pflag.Usage()

		os.Exit(1)
	}

	Logger.Println("Serving files from", root)

	// Explicitly handle signals. THis is needful to run as PID 1 in a container without
	// an init wrapper, which is literally the point of this program.
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	errs := make(chan error)

	server := http.Server{
		Addr:         ListenFlag,
		Handler:      static.Dir(root),
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: time.Minute,
		IdleTimeout:  time.Minute,
	}

	Logger.Println("Listening on", ListenFlag)
	go func() { errs <- server.ListenAndServe() }()

	select {
	case <-ctx.Done():
		Logger.Println("Shutting down")

		// Get a new signal context to kill a slow shutdown
		ctx, _ = signal.NotifyContext(context.Background(), os.Interrupt)
		server.Shutdown(ctx)
	case err := <-errs:
		Logger.Println("Listener error", err)
		os.Exit(1)
	}
}
