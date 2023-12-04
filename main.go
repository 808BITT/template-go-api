package main

import (
	"api/del"
	"api/get"
	"api/post"
	"api/put"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var port = ":8080"            // TODO: make this configurable
var assetPath = `..\dist\src` // TODO: make this configurable

var wg sync.WaitGroup
var stopFlag = new(bool)

func main() {
	runAPI()
}

func webApp(w http.ResponseWriter, r *http.Request) {
	p := assetPath + `\index.html`
	http.ServeFile(w, r, p)
}

func serveAsset(w http.ResponseWriter, r *http.Request) {
	p := assetPath + r.URL.Path
	http.ServeFile(w, r, p)
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	if *stopFlag {
		http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		get.Handler(w, r)
	case http.MethodPost:
		post.Handler(w, r)
	case http.MethodPut:
		put.Handler(w, r)
	case http.MethodDelete:
		del.Handler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func runAPI() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	http.HandleFunc("/", webApp)
	http.HandleFunc("/index.html", webApp)
	http.HandleFunc("/home", webApp)
	http.HandleFunc("/assets/{rest:.*}", serveAsset)
	http.HandleFunc("/api/", apiHandler)

	srv := &http.Server{
		Addr:         port,
		Handler:      nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Listening on port " + port + "...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server error:", err)
		}
	}()

	<-signalChan
	*stopFlag = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown Failed:", err)
	}

	wg.Wait()
	log.Println("Server exited")
}
