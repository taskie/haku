package haku

import (
	"io"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

const wsview = `
<html>
	<head>
		<script type="text/javascript">
			(function () {
				var s = '';
				var ws = new WebSocket('ws://localhost:8900/ws/');
				ws.addEventListener('message', function (ev) {
					s += ev.data;
					var out = document.getElementById("out");
					if (out != null) {
						out.textContent = s;
					}
				});
			})();
		</script>
	</head>
	<body>
		<pre id="out">
		</pre>
	</body>
</html>
`

func (h *Haku) generateWebSocketServer() (func(ws *websocket.Conn), error) {
	r, err := h.generateReader()
	if err != nil {
		return nil, err
	}
	return func(ws *websocket.Conn) {
		writers := []io.Writer{ws}
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

func (h *Haku) runWebSocketServer() error {
	mux := http.ServeMux{}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(wsview))
	})
	wss, err := h.generateWebSocketServer()
	if err != nil {
		return err
	}
	mux.Handle("/ws/", websocket.Handler(wss))
	server := http.Server{
		Addr:    h.Addr,
		Handler: &mux,
	}
	return server.ListenAndServe()
}
