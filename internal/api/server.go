package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/benisong/bitchat/farp/identity"
	"github.com/benisong/bitchat/farp/ledger"
	"github.com/benisong/bitchat/farp/ratelimit"
)

// Server 本地 HTTP API 端点
// 仅绑定 127.0.0.1，提供 CLI/GUI 调用

type Server struct {
	addr    string
	node    *identity.Node
	quota   *ratelimit.Manager
	credits *ledger.CreditRecord // 本地视角的自己积分
}

func NewServer(node *identity.Node, quota *ratelimit.Manager, credits *ledger.CreditRecord) *Server {
	return &Server{
		node:    node,
		quota:   quota,
		credits: credits,
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

	// 打开随机本场端口
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return "", err
	}
	s.addr = l.Addr().String()
	go http.Serve(l, s.withCORS(mux))
	return s.addr, nil
}

func (s *Server) Addr() string { return s.addr }

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleStatus GET /status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	tokens, cap := s.quota.Peek(s.node.ID)
	resp := map[string]any{
		"farp_id":       s.node.ID,
		"epoch":         s.node.Epoch,
		"device_id":     s.node.DeviceID,
		"quota_tokens":  tokens,
		"quota_cap":     cap,
		"frozen":        s.credits.Frozen,
		"balance":       s.credits.Balance,
		"online_time":   time.Now().UTC().Unix(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleContacts GET /contacts
func (s *Server) handleContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// TODO: 读取 SQLite contacts 表
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"contacts": []any{}})
}

// handleMessages GET /messages/:pubkey
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// path 剈析： /messages/abc123
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"pubkey": parts[2], "limit": limit, "messages": []any{}})
}

// handleCredits GET /credits
func (s *Server) handleCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	resp := map[string]any{
		"pubkey":       s.node.ID,
		"balance":      s.credits.Balance,
		"frozen":       s.credits.Frozen,
		"contribution": fmt.Sprintf("%.2f%%", float64(s.credits.ContributionRatio)/100.0),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleRoutes GET /routes
func (s *Server) handleRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// TODO: 读取 routes 表
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"routes": []any{}})
}

// handleHallOfFame GET /halloffame
func (s *Server) handleHallOfFame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// TODO: 读取 hall_of_fame 表
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"hall_of_fame": []any{}})
}

// handleOutbox GET /outbox
func (s *Server) handleOutbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	// TODO: 读取 outbox 表牌
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
}
