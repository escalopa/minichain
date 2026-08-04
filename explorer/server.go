package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// Server exposes the cached chain as JSON API + HTML pages.
type Server struct {
	store *Store
}

func NewServer(store *Store) *Server {
	return &Server{store: store}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// HTML pages.
	mux.HandleFunc("GET /{$}", s.handleHome)
	mux.HandleFunc("GET /block/{ref}", s.handleBlock)
	mux.HandleFunc("GET /address/{addr}", s.handleAddress)
	mux.HandleFunc("GET /search", s.handleSearch)

	// JSON API.
	mux.HandleFunc("GET /api/blocks", s.handleAPIBlocks)
	mux.HandleFunc("GET /api/blocks/{ref}", s.handleAPIBlock)
	mux.HandleFunc("GET /api/address/{addr}", s.handleAPIAddress)

	return mux
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index", map[string]any{
		"Height": s.store.Height(),
		"Blocks": s.store.Recent(25),
	})
}

func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	ref := r.PathValue("ref")
	block, ok := s.store.BlockByRef(ref)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		s.render(w, "notfound", map[string]any{"Query": ref})
		return
	}
	s.render(w, "block", map[string]any{"Block": block})
}

func (s *Server) handleAddress(w http.ResponseWriter, r *http.Request) {
	addr := r.PathValue("addr")
	balance, history := s.store.AddressInfo(addr)
	s.render(w, "address", map[string]any{
		"Address": addr,
		"Balance": balance,
		"History": history,
	})
}

// handleSearch routes a free-form query to the right page:
// a known block (by index or hash) wins, anything else is an address.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if _, ok := s.store.BlockByRef(q); ok {
		http.Redirect(w, r, "/block/"+url.PathEscape(q), http.StatusFound)
		return
	}
	if q != "" {
		http.Redirect(w, r, "/address/"+url.PathEscape(q), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleAPIBlocks(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"height": s.store.Height(),
		"blocks": s.store.Recent(limit),
	})
}

func (s *Server) handleAPIBlock(w http.ResponseWriter, r *http.Request) {
	block, ok := s.store.BlockByRef(r.PathValue("ref"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "block not found"})
		return
	}
	writeJSON(w, http.StatusOK, block)
}

func (s *Server) handleAPIAddress(w http.ResponseWriter, r *http.Request) {
	addr := r.PathValue("addr")
	balance, history := s.store.AddressInfo(addr)
	writeJSON(w, http.StatusOK, map[string]any{
		"address": addr,
		"balance": balance,
		"history": history,
	})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode json: %v", err)
	}
}
