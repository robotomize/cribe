package botstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrEventRejected   = errors.New("event rejected")
	ErrSessionNotFound = errors.New("not found")
	ErrStateNotFound   = errors.New("not found")
)

type Backend interface {
	Get(ctx context.Context, k string) ([]byte, error)
	Set(ctx context.Context, k string, v []byte) error
	Delete(ctx context.Context, k string) error
}

type StateEncoded struct {
	Current  StateType `json:"current"`
	Previous StateType `json:"previous"`
}

type Options struct{}

func NewSession(identity string, backend Backend, state *StateMachine) *Session {
	return &Session{identity: identity, backend: backend, StateMachine: state}
}

type Session struct {
	*StateMachine
	identity string
	backend  Backend
}

func (s *Session) Clean(ctx context.Context) error {
	if err := s.backend.Delete(ctx, s.identity); err != nil {
		return fmt.Errorf("unable delete state: %w", err)
	}

	return nil
}

// Flush save fsm state to backend
func (s *Session) Flush(ctx context.Context) error {
	encoded, err := json.Marshal(StateEncoded{Current: s.curr, Previous: s.prev})
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err = s.backend.Set(ctx, s.identity, encoded); err != nil {
		return fmt.Errorf("unable flush state: %w", err)
	}

	return nil
}

// Load method load session data from backend and current state
func (s *Session) Load(ctx context.Context) error {
	var state StateEncoded
	// load state from backend
	bytes, err := s.backend.Get(ctx, s.identity)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			// load default state
			s.curr = Default

			return nil
		}

		return fmt.Errorf("get state from session: %w", err)
	}

	// unmarshal state
	if err = json.Unmarshal(bytes, &state); err != nil {
		return fmt.Errorf("unable marshal: %w", err)
	}

	// set state from backend
	s.curr = state.Current
	s.prev = state.Previous

	return nil
}
