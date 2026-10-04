package usecase

import (
	"context"
	"sync"
)

type ChatEvent struct {
	AccountID string      `json:"account_id"`
	PageID    string      `json:"page_id"`
	SenderID  string      `json:"sender_id"`
	Message   string      `json:"message"`
	Type      string      `json:"type"` // "text", "image", etc.
	Timestamp string      `json:"timestamp"`
	Platform  string      `json:"platform"` // "facebook" or "whatsapp" or "system"
}

type ChatRepository interface {
	SaveMessage(ctx context.Context, msg ChatEvent) error
	GetRecentMessages(ctx context.Context, accountID string, limit int) ([]ChatEvent, error)
}

type ChatStream struct {
	mu       sync.RWMutex
	clients  map[string]map[chan ChatEvent]bool
	repo     ChatRepository
}

func NewChatStream(repo ChatRepository) *ChatStream {
	return &ChatStream{
		clients: make(map[string]map[chan ChatEvent]bool),
		repo:    repo,
	}
}

func (s *ChatStream) Subscribe(accountID string) chan ChatEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, ok := s.clients[accountID]; !ok {
		s.clients[accountID] = make(map[chan ChatEvent]bool)
	}
	
	ch := make(chan ChatEvent, 10)
	s.clients[accountID][ch] = true
	return ch
}

func (s *ChatStream) Unsubscribe(accountID string, ch chan ChatEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if clients, ok := s.clients[accountID]; ok {
		delete(clients, ch)
		close(ch)
		if len(clients) == 0 {
			delete(s.clients, accountID)
		}
	}
}

func (s *ChatStream) Broadcast(accountID string, event ChatEvent) {
	// Save to DB first
	if s.repo != nil {
		go s.repo.SaveMessage(context.Background(), event)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if clients, ok := s.clients[accountID]; ok {
		for ch := range clients {
			select {
			case ch <- event:
			default:
				// Channel is full, drop message
			}
		}
	}
}

func (s *ChatStream) GetHistory(ctx context.Context, accountID string) ([]ChatEvent, error) {
	if s.repo != nil {
		return s.repo.GetRecentMessages(ctx, accountID, 500)
	}
	return []ChatEvent{}, nil
}
