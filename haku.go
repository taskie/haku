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
		out, err := exec.Command(h.Command[0], h.Command[1:]...).Output() // StdoutPipe
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(os.Stderr, err)
			return
		}
		w.WriteHeader(200)
		_, err = w.Write([]byte(out))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
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
	Config     []string `short:"c" long:"config" env:"HAKU_CONFIG"`
	Addr       string   `short:"a" long:"addr" env:"HAKU_ADDR"`
	Persistent bool     `short:"p" long:"persistent" env:"HAKU_PERSISTENT"`
	NoColor    bool     `long:"no-color" env:"NO_COLOR"`
	Verbose    bool     `short:"v" long:"verbose"`
	Version    bool     `short:"V" long:"version"`
}

func (o *Options) Canonicalize() error {
	if o.Addr == "" {
		o.Addr = ":8900"
	}
	return nil
}

func Main() {
	var opts Options
	args, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}
	_ = opts.Canonicalize()
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
