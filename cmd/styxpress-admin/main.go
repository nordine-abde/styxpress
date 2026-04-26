package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

//go:embed all:web
var webFiles embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "admin server listen address")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("styxpress-admin listening on http://%s", listener.Addr())

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", embeddedSPA())

	return mux
}

func embeddedSPA() http.Handler {
	dist, err := fs.Sub(webFiles, "web/dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "admin frontend is not built; run npm run build in admin/web", http.StatusServiceUnavailable)
		})
	}

	files := http.FS(dist)
	fileServer := http.FileServer(files)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		file, err := files.Open(path)
		if err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		r = r.Clone(r.Context())
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: styxpress-admin [flags]\n\n")
		flag.PrintDefaults()
	}
}
