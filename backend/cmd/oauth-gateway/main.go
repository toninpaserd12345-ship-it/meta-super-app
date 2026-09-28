package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type referral struct {
	Source string `json:"source"`
	Type   string `json:"type"`
	Ref    string `json:"ref"`
	AdID   string `json:"ad_id"`
}

type messagingEvent struct {
	Sender struct {
		ID string `json:"id"`
	} `json:"sender"`
	Message struct {
		MID      string    `json:"mid"`
		Referral *referral `json:"referral"`
	} `json:"message"`
	Referral *referral `json:"referral"`
	Postback struct {
		Referral *referral `json:"referral"`
	} `json:"postback"`
}

func eventReferral(event messagingEvent) *referral {
	if event.Message.Referral != nil {
		return event.Message.Referral
	}
	if event.Referral != nil {
		return event.Referral
	}
	return event.Postback.Referral
}

func main() {
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/meta/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		upstream := "http://127.0.0.1:8080/api/v1/meta/oauth/callback?" + r.URL.RawQuery
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream, nil)
		if err != nil {
			http.Error(w, "invalid callback", http.StatusBadRequest)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "callback unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if location := resp.Header.Get("Location"); location != "" {
			w.Header().Set("Location", location)
		}
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, io.LimitReader(resp.Body, 64<<10))
	})
	mux.HandleFunc("/api/v1/meta/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		upstream := "http://127.0.0.1:8080/api/v1/meta/webhook"
		if r.URL.RawQuery != "" {
			upstream += "?" + r.URL.RawQuery
		}
		req, err := http.NewRequestWithContext(r.Context(), r.Method, upstream, bytes.NewReader(body))
		if err != nil {
			http.Error(w, "invalid webhook", http.StatusBadRequest)
			return
		}
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		req.Header.Set("X-Hub-Signature-256", r.Header.Get("X-Hub-Signature-256"))
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "webhook unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var event struct {
				Object string `json:"object"`
				Entry  []struct {
					ID        string           `json:"id"`
					Messaging []messagingEvent `json:"messaging"`
					Changes   []struct {
						Field string `json:"field"`
						Value struct {
							PostID    string `json:"post_id"`
							CommentID string `json:"comment_id"`
						} `json:"value"`
					} `json:"changes"`
				} `json:"entry"`
			}
			if json.Unmarshal(body, &event) == nil {
				for _, entry := range event.Entry {
					fields := make([]string, 0, len(entry.Changes))
					for _, change := range entry.Changes {
						fields = append(fields, change.Field)
						if change.Value.PostID != "" || change.Value.CommentID != "" {
							log.Printf("WEBHOOK feed page_id=%s field=%s post_id=%s comment_id=%s", entry.ID, change.Field, change.Value.PostID, change.Value.CommentID)
						}
					}
					for _, message := range entry.Messaging {
						ref := eventReferral(message)
						if ref == nil {
							log.Printf("WEBHOOK message page_id=%s sender_id=%s message_id=%s attribution=direct", entry.ID, message.Sender.ID, message.Message.MID)
							continue
						}
						log.Printf("WEBHOOK message page_id=%s sender_id=%s message_id=%s referral_source=%s referral_type=%s ad_id=%s ref=%s", entry.ID, message.Sender.ID, message.Message.MID, ref.Source, ref.Type, ref.AdID, ref.Ref)
					}
					log.Printf("WEBHOOK accepted object=%s page_id=%s fields=%v messaging_events=%d bytes=%d", event.Object, entry.ID, fields, len(entry.Messaging), len(body))
				}
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, io.LimitReader(resp.Body, 64<<10))
	})
	server := &http.Server{Addr: "127.0.0.1:8090", Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	log.Printf("OAuth callback gateway listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
