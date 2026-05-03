// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

const chatMaxAge = 8 * time.Hour

var (
	// ErrChatNotFound indicates no chat history exists for the given user.
	ErrChatNotFound = errors.New("chat not found")
	// ErrChatExpired indicates the chat history has passed its TTL.
	ErrChatExpired = errors.New("chat expired")
)

// ChatSession holds persisted chat history keyed by user email.
type ChatSession struct {
	Messages  json.RawMessage `json:"messages"`
	TaskID    string          `json:"taskId"`
	UpdatedAt int64           `json:"-"`
}

// ChatStore persists chat history per authenticated user.
type ChatStore interface {
	PutChat(ctx context.Context, email string, chat ChatSession) error
	GetChat(ctx context.Context, email string) (*ChatSession, error)
	DeleteChat(ctx context.Context, email string) error
}

// MemoryChatStore is a concurrency-safe in-memory chat store.
type MemoryChatStore struct {
	mu   sync.RWMutex
	chat map[string]ChatSession
}

// NewMemoryChatStore creates an in-memory chat store.
func NewMemoryChatStore() *MemoryChatStore {
	return &MemoryChatStore{chat: make(map[string]ChatSession)}
}

func (m *MemoryChatStore) PutChat(_ context.Context, email string, chat ChatSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	chat.UpdatedAt = time.Now().Unix()
	m.chat[email] = chat
	return nil
}

func (m *MemoryChatStore) GetChat(_ context.Context, email string) (*ChatSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	chat, ok := m.chat[email]
	if !ok {
		return nil, ErrChatNotFound
	}
	if time.Now().Unix() > chat.UpdatedAt+int64(chatMaxAge.Seconds()) {
		return nil, ErrChatExpired
	}
	return &chat, nil
}

func (m *MemoryChatStore) DeleteChat(_ context.Context, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.chat, email)
	return nil
}
