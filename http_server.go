package haku

import (
	"io"
	"net/http"
	"os"
)

func (h *Haku) generateHTTPHandlerFunc() (func(http.ResponseWriter, *http.Request), error) {
	r, err := h.generateReader()
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, req *http.Request) {
		writers := []io.Writer{w}
		if p, ok := r.(*Persister); ok && p.Status == PersisterStatusRead {
			pb := p.CreatePseudoBuffer()
			io.Copy(io.MultiWriter(writers...), &pb)
		} else {
			if h.Mode == HakuModeTee {
				writers = append(writers, os.Stdout)
			}
			io.Copy(io.MultiWriter(writers...), r)
		}
	}, nil
}

func (h *Haku) runHTTPServer() error {
	mux := http.ServeMux{}
	hf, err := h.generateHTTPHandlerFunc()
	if err != nil {
		return err
	}
	mux.HandleFunc("/", hf)
	server := http.Server{
		Addr:    h.Addr,
		Handler: &mux,
	}
	return server.ListenAndServe()
}
