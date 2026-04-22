// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// ServerSession holds session data server-side, keeping sensitive tokens
// out of the client cookie.
type ServerSession struct {
	AccessToken string
	Login       string
	Name        string
	AvatarURL   string
	Email       string
	Groups      []string
	ExpiresAt   int64
}

// SessionStore abstracts server-side session storage so the implementation
// can be swapped from in-memory to Redis/Valkey without code changes.
type SessionStore interface {
	Put(ctx context.Context, id string, sess ServerSession) error
	Get(ctx context.Context, id string) (*ServerSession, error)
	Delete(ctx context.Context, id string) error
}

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

// MemorySessionStore is a concurrency-safe in-memory session store with
// passive TTL expiration. Suitable for single-replica deployments.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]ServerSession
	chat     map[string]ChatSession
}

// NewMemorySessionStore creates an in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]ServerSession),
		chat:     make(map[string]ChatSession),
	}
}

func (m *MemorySessionStore) Put(_ context.Context, id string, sess ServerSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = sess
	return nil
}

func (m *MemorySessionStore) Get(_ context.Context, id string) (*ServerSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if time.Now().Unix() > sess.ExpiresAt {
		return nil, ErrSessionExpired
	}
	return &sess, nil
}

func (m *MemorySessionStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *MemorySessionStore) PutChat(_ context.Context, email string, chat ChatSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	chat.UpdatedAt = time.Now().Unix()
	m.chat[email] = chat
	return nil
}

func (m *MemorySessionStore) GetChat(_ context.Context, email string) (*ChatSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	chat, ok := m.chat[email]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if time.Now().Unix() > chat.UpdatedAt+int64(sessionMaxAge.Seconds()) {
		return nil, ErrSessionExpired
	}
	return &chat, nil
}

func (m *MemorySessionStore) DeleteChat(_ context.Context, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.chat, email)
	return nil
}

// Len returns the number of stored sessions (including expired). Used in tests.
func (m *MemorySessionStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
