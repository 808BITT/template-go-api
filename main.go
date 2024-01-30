package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var port = ":8080"            // TODO: make this configurable
var assetPath = `..\dist\src` // TODO: make this configurable

var wg sync.WaitGroup
var stopFlag = new(bool)

func webApp(w http.ResponseWriter, r *http.Request) {
	p := assetPath + r.URL.Path
	log.Println("webApp: " + p)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		http.ServeFile(w, r, assetPath+"/index.html")
	} else {
		http.ServeFile(w, r, p)
	}
}

func serveAsset(w http.ResponseWriter, r *http.Request) {
	assetPath := assetPath + r.URL.Path
	http.ServeFile(w, r, assetPath)
}

func runAPI() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	r := mux.NewRouter()
	r.HandleFunc("/", webApp)
	r.HandleFunc("/assets/{rest:.*}", serveAsset)

	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"*"},
	}).Handler(r)

	srv := &http.Server{
		Handler: handler,
		Addr:    port,
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

func main() {
	runAPI()
}
