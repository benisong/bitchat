package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/benisong/bitchat/farp/contribution"
	"github.com/benisong/bitchat/farp/identity"
	"github.com/benisong/bitchat/farp/ratelimit"
)

// Server 是仅供本机 CLI/GUI 调用的 HTTP API。
type Server struct {
	addr    string
	node    *identity.Node
	quota   *ratelimit.Manager
	credits *contribution.State
	server  *http.Server
	started time.Time
}

func NewServer(node *identity.Node, quota *ratelimit.Manager, credits *contribution.State) *Server {
	return &Server{
		node:    node,
		quota:   quota,
		credits: credits,
		started: time.Now().UTC(),
	}
}

func (s *Server) Start() (string, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/contacts", s.handleContacts)
	mux.HandleFunc("/messages/", s.handleMessages)
	mux.HandleFunc("/credits", s.handleCredits)
	mux.HandleFunc("/routes", s.handleRoutes)
	mux.HandleFunc("/halloffame", s.handleHallOfFame)
	mux.HandleFunc("/outbox", s.handleOutbox)

	// 使用随机本机端口，避免暴露到局域网。
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return "", err
	}
	s.addr = l.Addr().String()
	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		if err := s.server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("local API stopped: %v", err)
		}
	}()
	return s.addr, nil
}

func (s *Server) Addr() string { return s.addr }

func (s *Server) Close(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// handleStatus GET /status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	tokens, cap := s.quota.Peek(s.node.ID)
	credits := s.credits.Snapshot()
	resp := map[string]any{
		"farp_id":               s.node.ID,
		"epoch":                 s.node.Epoch(),
		"device_id":             s.node.DeviceID,
		"quota_tokens":          tokens,
		"quota_cap":             cap,
		"spendable_credit":      credits.SpendableCredit,
		"lifetime_contribution": credits.LifetimeContribution,
		"burned_credit":         credits.BurnedCredit,
		"started_at":            s.started.Unix(),
		"uptime_seconds":        int64(time.Since(s.started).Seconds()),
	}
	writeJSON(w, resp)
}

// handleContacts GET /contacts
func (s *Server) handleContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// TODO: 读取 SQLite contacts 表
	writeJSON(w, map[string]any{"contacts": []any{}})
}

// handleMessages GET /messages/:pubkey
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// 路径格式：/messages/abc123
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "need pubkey", http.StatusBadRequest)
		return
	}
	_ = parts[2] // pubkey
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	// TODO: 读取 messages 表分页
	writeJSON(w, map[string]any{"pubkey": parts[2], "limit": limit, "messages": []any{}})
}

// handleCredits GET /credits
func (s *Server) handleCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	credits := s.credits.Snapshot()
	writeJSON(w, map[string]any{
		"pubkey":                s.node.ID,
		"state_version":         credits.Version,
		"spendable_credit":      credits.SpendableCredit,
		"lifetime_contribution": credits.LifetimeContribution,
		"burned_credit":         credits.BurnedCredit,
		"updated_at":            credits.UpdatedAt,
	})
}

// handleRoutes GET /routes
func (s *Server) handleRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// TODO: 读取 routes 表
	writeJSON(w, map[string]any{"routes": []any{}})
}

// handleHallOfFame GET /halloffame
func (s *Server) handleHallOfFame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// TODO: 读取 hall_of_fame 表
	writeJSON(w, map[string]any{"hall_of_fame": []any{}})
}

// handleOutbox GET /outbox
func (s *Server) handleOutbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// TODO: 读取 outbox 表
	writeJSON(w, map[string]any{"items": []any{}})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write local API response: %v", err)
	}
}
