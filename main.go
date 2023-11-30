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
	"golang.org/x/sys/windows/svc"
)

var port = ":8080"          // TODO: make this configurable
var assetPath = `C:\public` // TODO: make this configurable

var wg sync.WaitGroup
var stopFlag = new(bool)

type myService struct{}

func (m *myService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	go func() {
		runAPI()
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	for c := range r {
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			*stopFlag = true
			changes <- svc.Status{State: svc.StopPending}
			return
		default:
			log.Println("Command not recognized:", c)
		}
	}
	return
}

func webApp(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, assetPath+`\index.html`)
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

func runService() {
	handler := &myService{}
	err := svc.Run("ETL API", handler)
	if err != nil {
		log.Fatalf("Service failed: %v", err)
	}
	log.Println("Service exited.")
}

func main() {
	isWinServ, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running as a windows service: %v", err)
	}

	if isWinServ {
		runService()
		return
	}
	runAPI()
}
