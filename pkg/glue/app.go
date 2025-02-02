package glue

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sjc5/kit/pkg/grace"
)

func (fw *Instance[AHD, SE, CEE]) App() {
	port := fw.GetEnv().Meta.Port
	r := fw.initRouter()

	server := &http.Server{
		Addr:                         fmt.Sprintf(":%d", port),
		Handler:                      http.TimeoutHandler(r, 60*time.Second, "Request timed out"),
		ReadTimeout:                  15 * time.Second,
		WriteTimeout:                 30 * time.Second,
		IdleTimeout:                  60 * time.Second,
		ReadHeaderTimeout:            10 * time.Second,
		MaxHeaderBytes:               1 << 20, // 1 MB
		DisableGeneralOptionsHandler: true,
		ErrorLog:                     log.New(os.Stderr, "HTTP: ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	grace.Orchestrate(grace.OrchestrateOptions{
		StartupCallback: func() error {
			log.Printf("Starting server on: http://localhost:%d\n", port)

			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server listen and serve error: %v\n", err)
			}

			return nil
		},
		ShutdownCallback: func(shutdownCtx context.Context) error {
			log.Printf("Shutting down server on: http://localhost:%d\n", port)

			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Fatalf("Server shutdown error: %v\n", err)
			}

			return nil
		},
	})
}
