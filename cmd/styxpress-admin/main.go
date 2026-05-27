package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nordine-abde/styxpress/internal/api"
)

//go:embed all:web
var webFiles embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "admin server listen address")
	configPath := flag.String("config", "", "admin config path")
	flag.Parse()

	var apiServer *api.Server
	var err error
	if *configPath == "" {
		apiServer, err = api.NewDefault(log.Default())
	} else {
		apiServer, err = api.New(*configPath, log.Default())
	}
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Handler:           newHandler(apiServer.Handler(), apiServer.Token()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("styxpress-admin listening on http://%s", listener.Addr())
	log.Printf("styxpress-admin API session token: %s", apiServer.Token())

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func newHandler(apiHandler http.Handler, sessionToken string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	mux.Handle("/", embeddedSPA(sessionToken))

	return mux
}

func embeddedSPA(sessionToken string) http.Handler {
	dist, err := fs.Sub(webFiles, "web/dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "admin frontend is not built; run npm run build in admin/web", http.StatusServiceUnavailable)
		})
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "admin frontend index.html is missing; run npm run build in admin/web", http.StatusServiceUnavailable)
		})
	}
	index = injectSessionBootstrap(index, sessionToken)

	files := http.FS(dist)
	fileServer := http.FileServer(files)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || path == "index.html" {
			serveIndex(w, r, index)
			return
		}

		file, err := files.Open(path)
		if err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		serveIndex(w, r, index)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, index []byte) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
}

func injectSessionBootstrap(index []byte, sessionToken string) []byte {
	tokenJSON, err := json.Marshal(sessionToken)
	if err != nil {
		tokenJSON = []byte(`""`)
	}
	script := []byte("<script>window.__STYXPRESS_SESSION__=" + string(tokenJSON) + ";</script>\n")
	headClose := []byte("</head>")
	if bytes.Contains(index, headClose) {
		return bytes.Replace(index, headClose, append(script, headClose...), 1)
	}
	return append(script, index...)
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: styxpress-admin [flags]\n\n")
		flag.PrintDefaults()
	}
}
