package statemachine

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const NoState = ""

type StateEvent func(sm *StateMachine)
type TimeoutEvent func(sm *StateMachine, eventType EventType)
type StateSelector func(state *State) string
type ErrorHandler func(err interface{}, eventType EventType)

type EventType int

const (
	EventEnter EventType = iota
	EventLeave
	EventState
)

type State struct {
	ID             string
	OnEnter        StateEvent
	OnEnterTimeout time.Duration
	OnLeave        StateEvent
	OnLeaveTimeout time.Duration
	Selector       StateSelector
	StateTimeout   time.Duration
}

type StateMachine struct {
	current        unsafe.Pointer
	states         map[string]*State
	advanceLock    sync.Mutex
	onTimeout      TimeoutEvent
	onError        ErrorHandler
	timeoutTracker chan struct{}
	timeoutLock    sync.Mutex
	eventLock      sync.Mutex
}

func NewStateMachine(timeoutEvent ...TimeoutEvent) *StateMachine {
	var event TimeoutEvent
	if len(timeoutEvent) > 0 {
		event = timeoutEvent[0]
	}
	return &StateMachine{
		states:         make(map[string]*State),
		onTimeout:      event,
		timeoutTracker: nil,
	}
}

func (sm *StateMachine) WithErrorHandler(eh ErrorHandler) {
	sm.onError = eh
}

func (sm *StateMachine) AddState(state *State) {
	sm.states[state.ID] = state
}

func (sm *StateMachine) Start(initialState string, triggerEvents ...bool) (bool, interface{}) {
	doTrigger := false
	if len(triggerEvents) > 0 {
		doTrigger = triggerEvents[0]
	}

	return sm.internalSwitch(initialState, doTrigger)
}

func (sm *StateMachine) runStateEvent(event StateEvent, timeout time.Duration, eventType EventType) interface{} {
	if event == nil {
		return nil
	}
	var errPtr unsafe.Pointer

	if timeout.Nanoseconds() == 0 {
		func() {
			defer func() {
				errLocal := recover()
				atomic.StorePointer(&errPtr, unsafe.Pointer(&errLocal))
				if errLocal != nil && sm.onError != nil {
					sm.onError(errLocal, eventType)
				}
			}()

			event(sm)
		}()
	} else {
		ch := make(chan struct{}, 1)
		go func() {
			defer func() {
				errLocal := recover()
				atomic.StorePointer(&errPtr, unsafe.Pointer(&errLocal))
				if errLocal != nil && sm.onError != nil {
					sm.onError(errLocal, eventType)
				}
			}()

			event(sm)
			_, ok := <-ch
			if ok {
				ch <- struct{}{}
			}
		}()

		select {
		case <-ch:
			close(ch)
		case <-time.After(timeout):
			close(ch)
			if sm.onTimeout != nil {
				sm.onTimeout(sm, eventType)
			}
		}
	}

	err := (*interface{})(atomic.LoadPointer(&errPtr))

	if err == nil {
		return nil
	}

	return *err
}

func (sm *StateMachine) leaveState(state *State, triggerEvents bool) interface{} {
	if state != nil && triggerEvents && state.OnLeave != nil {
		return sm.runStateEvent(state.OnLeave, state.OnLeaveTimeout, EventLeave)
	}

	return nil
}

func (sm *StateMachine) enterState(state *State, triggerEvents bool) (bool, interface{}) {
	var err interface{}
	atomic.StorePointer(&sm.current, unsafe.Pointer(state))
	if state != nil {
		if state.OnEnter != nil && triggerEvents {
			err = sm.runStateEvent(state.OnEnter, state.OnEnterTimeout, EventEnter)
		}
		if state.StateTimeout.Nanoseconds() > 0 {
			sm.timeoutLock.Lock()
			sm.timeoutTracker = make(chan struct{}, 1)
			sm.timeoutLock.Unlock()
			go func() {
				defer func() {
					sm.timeoutLock.Lock()
					close(sm.timeoutTracker)
					sm.timeoutTracker = nil
					sm.timeoutLock.Unlock()
				}()

				select {
				case <-sm.timeoutTracker:
				case <-time.After(state.StateTimeout):
					if sm.onTimeout != nil {
						sm.onTimeout(sm, EventState)
					}
				}
			}()
		}

		return true, err
	}

	return false, err
}

func (sm *StateMachine) internalSwitch(toState string, triggerEvents bool) (bool, interface{}) {
	if toState == NoState {
		return false, nil
	}

	sm.timeoutLock.Lock()
	if sm.timeoutTracker != nil {
		close(sm.timeoutTracker)
		sm.timeoutTracker = nil
	}
	sm.timeoutLock.Unlock()

	sm.eventLock.Lock()
	err := sm.leaveState(sm.CurrentState(), triggerEvents)
	sm.eventLock.Unlock()

	if err != nil {
		return false, err
	}

	nState, _ := sm.states[toState]

	result := false
	sm.eventLock.Lock()
	result, err = sm.enterState(nState, triggerEvents)
	sm.eventLock.Unlock()

	return result, err
}

func (sm *StateMachine) getNextState() string {
	advanceId := NoState
	current := sm.CurrentState()

	if current != nil && current.Selector != nil {
		advanceId = current.Selector(current)
	}

	return advanceId
}

func (sm *StateMachine) Advance() (bool, interface{}) {
	sm.advanceLock.Lock()
	defer sm.advanceLock.Unlock()

	if sm.current == nil {
		return false, nil
	}

	return sm.internalSwitch(sm.getNextState(), true)
}

func (sm *StateMachine) CurrentState() *State {
	ptr := atomic.LoadPointer(&sm.current)
	if ptr == nil {
		return nil
	}

	return (*State)(ptr)
}

func (sm *StateMachine) EmergencySwitch(stateId string, triggerEvents ...bool) (bool, interface{}) {
	sm.advanceLock.Lock()
	defer sm.advanceLock.Unlock()

	return sm.internalSwitch(stateId, len(triggerEvents) > 0 && triggerEvents[0])
}

func (sm *StateMachine) AutoAdvance(tryPeriod time.Duration, terminalStates []string) {
	for {
		result, err := sm.Advance()
		if err != nil {
			// stop state machine
			return
		}
		if result {
			cs := sm.CurrentState().ID
			for _, ts := range terminalStates {
				if cs == ts {
					// state machine has reached one of terminal states, stop it
					return
				}
			}
		} else {
			// cannot advance yet, wait
			time.Sleep(tryPeriod)
		}
	}
}
