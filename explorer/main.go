package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	nodeURL := envOr("NODE_URL", "http://localhost:3000")
	port := envOr("PORT", "8080")
	pollEvery := 2 * time.Second

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := NewStore()
	client := NewNodeClient(nodeURL)
	go poll(ctx, client, store, pollEvery)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: NewServer(store).Handler(),
	}
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

// poll keeps the store in sync with the node. Fetching the whole chain
// every tick is fine for an educational network; a real explorer would
// track the tip and backfill.
func poll(ctx context.Context, client *NodeClient, store *Store, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		blocks, err := client.Blocks(ctx)
		switch {
		case err != nil:
			log.Printf("poll node: %v", err)
		case len(blocks) != store.Height():
			store.Update(blocks)
			log.Printf("synced %d blocks", len(blocks))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
