// The composition root: the only place that knows about concrete
// adapters and wires them into the core's ports.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/escalopa/minichain/explorer/internal/adapter/httpserver"
	"github.com/escalopa/minichain/explorer/internal/adapter/memstore"
	"github.com/escalopa/minichain/explorer/internal/adapter/nodeclient"
	"github.com/escalopa/minichain/explorer/internal/core/service"
)

func main() {
	nodeURL := envOr("NODE_URL", "http://localhost:3000")
	port := envOr("PORT", "8080")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The wiring: concrete adapters are created here and immediately
	// erased to their port interfaces as they enter the core. This is
	// the only file that names both sides — everything else depends
	// on abstractions.
	repo := memstore.New()
	source := nodeclient.New(nodeURL)
	syncer := service.NewSyncer(source, repo, 2*time.Second)
	explorer := service.NewExplorer(repo)

	// The syncer owns the only writer to the repository; every HTTP
	// handler is a reader. That, plus the RWMutex inside memstore, is
	// the whole concurrency story of this service.
	go syncer.Run(ctx)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: httpserver.New(explorer).Handler(),
	}
	// Graceful shutdown: on SIGINT/SIGTERM stop accepting connections
	// and give in-flight requests up to 5s to finish. The shutdown
	// context must be a fresh Background one — ctx is already
	// cancelled at this point and would abort immediately.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("explorer listening on :%s, node at %s", port, nodeURL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
