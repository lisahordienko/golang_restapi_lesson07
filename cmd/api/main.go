package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homework/internal/app"
)

type serverConfig struct {
	scheme   string
	tls      bool
	certFile string
	keyFile  string
}

func loadServerConfig(certFile, keyFile string) (serverConfig, error) {
	if certFile == "" && keyFile == "" {
		return serverConfig{scheme: "http"}, nil
	}
	if certFile == "" || keyFile == "" {
		return serverConfig{}, errors.New("TLS_CERT_FILE and TLS_KEY_FILE must be configured together")
	}
	return serverConfig{
		scheme:   "https",
		tls:      true,
		certFile: certFile,
		keyFile:  keyFile,
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	config, err := loadServerConfig(os.Getenv("TLS_CERT_FILE"), os.Getenv("TLS_KEY_FILE"))
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           app.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		if config.tls {
			serverErrors <- server.ListenAndServeTLS(config.certFile, config.keyFile)
			return
		}
		serverErrors <- server.ListenAndServe()
	}()

	log.Printf("listening over %s on %s", config.scheme, addr)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("server shutdown failed: %v", err)
		}
		log.Println("server stopped")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}
