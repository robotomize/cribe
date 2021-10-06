package botstate

import (
	"sync"
)

const (
	Default StateType = "init"
	Noop    EventType = "noop"
)

type StateType string

type EventType string

type EventContext interface{}

type Action interface {
	Execute(eventCtx EventContext) EventType
}

type Events map[EventType]StateType

type State struct {
	Action Action
	Events Events
}

type States map[StateType]State

func NewStateMachine(states States) *StateMachine {
	return &StateMachine{states: states}
}

type StateMachine struct {
	mtx    sync.RWMutex
	prev   StateType
	curr   StateType
	states States
}

func (s *StateMachine) Current() StateType {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.curr
}

func (s *StateMachine) Previous() StateType {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.prev
}

func (s *StateMachine) SendEvent(event EventType, eventCtx EventContext) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for {
		nextState, err := s.getNextState(event)
		if err != nil {
			return ErrEventRejected
		}

		state, ok := s.states[nextState]
		if !ok {
			return ErrStateNotFound
		}

		s.prev = s.curr
		s.curr = nextState

		nextEvent := state.Action.Execute(eventCtx)
		if nextEvent == Noop {
			return nil
		}

		event = nextEvent
	}
}

func (s *StateMachine) getNextState(event EventType) (StateType, error) {
	if state, ok := s.states[s.curr]; ok {
		if state.Events != nil {
			if next, ok := state.Events[event]; ok {
				return next, nil
			}
		}
	}

	return Default, ErrEventRejected
}
