package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/benisong/bitchat/farp/contribution"
	"github.com/benisong/bitchat/farp/identity"
	"github.com/benisong/bitchat/farp/ratelimit"
)

func TestLocalAPIStatusDoesNotEnableBrowserCORS(t *testing.T) {
	node, err := identity.Generate()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	credits := contribution.NewState(node.ID)
	if err := credits.Earn(7); err != nil {
		t.Fatalf("earn credit: %v", err)
	}

	server := NewServer(node, ratelimit.NewManager(), credits)
	addr, err := server.Start()
	if err != nil {
		t.Fatalf("start API: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Close(ctx); err != nil {
			t.Errorf("close API: %v", err)
		}
	})

	request, err := http.NewRequest(http.MethodGet, "http://"+addr+"/status", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Origin", "https://untrusted.example")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request status: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if value := response.Header.Get("Access-Control-Allow-Origin"); value != "" {
		t.Fatalf("unexpected browser CORS permission %q", value)
	}

	var status struct {
		FARPID               string `json:"farp_id"`
		SpendableCredit      int64  `json:"spendable_credit"`
		LifetimeContribution int64  `json:"lifetime_contribution"`
	}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.FARPID != node.ID || status.SpendableCredit != 7 || status.LifetimeContribution != 7 {
		t.Fatalf("unexpected status payload: %+v", status)
	}
}
