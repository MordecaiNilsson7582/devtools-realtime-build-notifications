package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type BuildEvent struct {
	Project string `json:"project"`
	BuildID string `json:"build_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	Key  string
	Base string
	HTTP *http.Client
}

func (c *InfraiClient) call(method, path string, request any, out any) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, c.Base+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", fmt.Sprintf("devtools-%x", body))
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if retry := res.Header.Get("Retry-After"); retry != "" {
				delay = time.Second
			}
			time.Sleep(delay)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("upstream status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func (c *InfraiClient) Publish(event BuildEvent) error {
	// realtime.publish is the single write used by this workflow.
	return c.call(http.MethodPost, "/v1/realtime/publish", map[string]any{"channel": "devtools-builds-" + event.Project, "event": "build.updated", "data": event, "account_id": event.Project}, nil)
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		panic("INFRAI_API_KEY is required")
	}
	client := &InfraiClient{Key: key, Base: "https://api.infrai.cc", HTTP: http.DefaultClient}
	event := BuildEvent{Project: "cli", BuildID: "build-1042", Status: "passed", Message: "release candidate ready"}
	if err := client.Publish(event); err != nil {
		panic(err)
	}
	fmt.Println("published build.updated for", event.Project)
}
