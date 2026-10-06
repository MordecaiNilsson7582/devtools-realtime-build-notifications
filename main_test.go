package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublishEnvelopeDecision(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"accepted", `{"ok":true,"data":{},"error":null,"metadata":{}}`, false},
		{"rejected", `{"ok":false,"data":null,"error":{"code":"INVALID_ARGUMENT"},"metadata":{}}`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c := &InfraiClient{Key: "test", Base: srv.URL, HTTP: srv.Client()}
			err := c.Publish(BuildEvent{Project: "cli", BuildID: "b1", Status: "passed"})
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
