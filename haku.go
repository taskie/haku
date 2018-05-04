package haku

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jessevdk/go-flags"
)

type Haku struct {
	Addr         string
	UseWebSocket bool
	Mode         string
	Persistent   bool
	Command      []string
}

const (
	HakuModeReader = "reader"
	HakuModeTee    = "tee"
	HakuModeExec   = "exec"
)

func (h *Haku) generatePersisterIfNeeded(r io.Reader) (io.Reader, error) {
	if h.Persistent {
		p := NewPersister(r)
		return &p, nil
	} else {
		return r, nil
	}
}

func (h *Haku) generateReader() (io.Reader, error) {
	switch h.Mode {
	case HakuModeExec:
		cmd := exec.Command(h.Command[0], h.Command[1:]...)
		fmt.Printf("1: %v\n", cmd)
		out, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		fmt.Printf("2: %v\n", cmd)
		go func() {
			err = cmd.Run()
			fmt.Printf("3: %v %s\n", cmd, err)
		}()
		return out, nil
	default:
		return h.generatePersisterIfNeeded(os.Stdin)
	}
}

func (h *Haku) Run() {
	var err error
	if h.UseWebSocket {
		err = h.runWebSocketServer()
	} else {
		err = h.runHTTPServer()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

type Options struct {
	// Config     []string `short:"c" long:"config" env:"HAKU_CONFIG" description:"config file"`
	Addr         string `short:"a" long:"addr" env:"HAKU_ADDR" default:"localhost:8900" value-name:"ADDR" description:"server address"`
	UseWebSocket bool   `short:"S" long:"web-socket" env:"HAKU_WEB_SOCKET" description:"use WebSocket"`
	Persistent   bool   `short:"p" long:"persistent" env:"HAKU_PERSISTENT" description:"persist stdin"`
	NoColor      bool   `long:"no-color" env:"NO_COLOR" description:"NOT colorize output"`
	Verbose      []bool `short:"v" long:"verbose" description:"verbose mode"`
	Version      bool   `short:"V" long:"version" description:"show version"`
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
			Addr:         opts.Addr,
			UseWebSocket: opts.UseWebSocket,
			Mode:         HakuModeTee,
			Persistent:   opts.Persistent,
		}
	} else {
		h = Haku{
			Addr:         opts.Addr,
			UseWebSocket: opts.UseWebSocket,
			Mode:         HakuModeExec,
			Command:      args,
		}
	}
	h.Run()
}
