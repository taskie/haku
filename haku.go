package haku

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
)

const (
	ModeReader = "reader"
	ModeTee    = "tee"
	ModeExec   = "exec"
)

type ExecCommandHandler struct {
	Command []string
}

func (h *ExecCommandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		cmd := exec.Command(h.Command[0], h.Command[1:]...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(os.Stderr, err)
			return
		}
		err = cmd.Start()
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(os.Stderr, err)
			return
		}
		w.WriteHeader(200)
		_, err = io.Copy(w, stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		err = cmd.Wait()
		if err != nil {
			// do nothing
		}
	} else {
		w.WriteHeader(405)
	}
}

type ReaderHandler struct {
	Reader     io.Reader
	Tee        bool
	Persistent bool
	Buffer     bytes.Buffer
	Bytes      []byte
	Status     string
	Mutex      sync.Mutex
}

const (
	ReaderHandlerStatusInitial = ""
	ReaderHandlerStatusRead    = "read"
)

func (h *ReaderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.WriteHeader(200)
		h.Mutex.Lock()
		defer h.Mutex.Unlock()
		writers := []io.Writer{w}
		if h.Status == ReaderHandlerStatusInitial {
			if h.Tee {
				writers = append(writers, os.Stdout)
			}
			if h.Persistent {
				writers = append(writers, &h.Buffer)
			}
		}
		dest := io.MultiWriter(writers...)
		src := h.Reader
		if h.Persistent && h.Status == ReaderHandlerStatusRead {
			src = &h.Buffer
			h.Buffer.Reset()
			h.Buffer.Write(h.Bytes)
		}
		_, err := io.Copy(dest, src)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if h.Status == ReaderHandlerStatusInitial {
			h.Status = ReaderHandlerStatusRead
			h.Bytes = h.Buffer.Bytes()
		}
	} else {
		w.WriteHeader(405)
	}
}

type Haku struct {
	Addr       string
	Mode       string
	Persistent bool
	Command    []string
}

func (h *Haku) ListenAndServe(server *http.Server) {
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (h *Haku) Run() {
	var handler http.Handler
	switch h.Mode {
	case ModeReader:
		handler = &ReaderHandler{
			Reader:     os.Stdin,
			Persistent: h.Persistent,
		}
	case ModeTee:
		handler = &ReaderHandler{
			Reader:     os.Stdin,
			Tee:        true,
			Persistent: h.Persistent,
		}
	case ModeExec:
		handler = &ExecCommandHandler{
			Command: h.Command,
		}
	}
	server := http.Server{
		Addr:    h.Addr,
		Handler: handler,
	}
	go h.ListenAndServe(&server)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM|syscall.SIGHUP|syscall.SIGINT)
	select {
	case s := <-signalChan:
		fmt.Fprintln(os.Stderr, s)
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		if err := server.Shutdown(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

type Options struct {
	// Config     []string `short:"c" long:"config" env:"HAKU_CONFIG" description:"config file"`
	Addr       string `short:"a" long:"addr" env:"HAKU_ADDR" default:":8900" value-name:"ADDR" description:"server address"`
	Persistent bool   `short:"p" long:"persistent" env:"HAKU_PERSISTENT" description:"persist stdin"`
	NoColor    bool   `long:"no-color" env:"NO_COLOR" description:"NOT colorize output"`
	Verbose    []bool `short:"v" long:"verbose" description:"verbose mode"`
	Version    bool   `short:"V" long:"version" description:"show version"`
}

var (
	Version = "0.0.1"
)

func Main(args []string) {
	var opts Options
	args, err := flags.ParseArgs(&opts, args)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
	if opts.Version {
		fmt.Println(Version)
		os.Exit(0)
	}
	var h Haku
	if len(args) == 0 {
		h = Haku{
			Addr:       opts.Addr,
			Mode:       ModeTee,
			Persistent: opts.Persistent,
		}
	} else {
		h = Haku{
			Addr:    opts.Addr,
			Mode:    ModeExec,
			Command: args,
		}
	}
	h.Run()
}
