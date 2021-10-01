package botstate

import (
	"encoding/json"
	"errors"
	"fmt"
)

type StateEncoded struct {
	State StateType `json:"state"`
}

func NewSession(identifier string, backend Backend, state *StateMachine) (*Session, error) {
	var current StateEncoded
	// load state from backend
	s := &Session{identifier: identifier, backend: backend, state: state}
	bytes, err := s.backend.Get(identifier)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// load default state
			s.state.curr = Default

			return s, nil
		}
		return nil, fmt.Errorf("get state from session: %w", err)
	}

	// unmarshal state
	if err = json.Unmarshal(bytes, &current); err != nil {
		return nil, fmt.Errorf("unable marshal: %w", err)
	}

	// set current state from backend
	s.state.curr = current.State

	return s, nil
}

type Session struct {
	state      *StateMachine
	identifier string
	backend    Backend
}
