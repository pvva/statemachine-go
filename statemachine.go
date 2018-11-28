package statemachine

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const NoState = ""

type StateEvent func(sm *StateMachine)
type TimeoutEvent func(sm *StateMachine, timeoutType TimeoutType)
type StateSelector func(state *State) string

type TimeoutType int

const (
	TimeoutEnter TimeoutType = iota
	TimeoutLeave
	TimeoutState
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
	timeoutTracker chan struct{}
	timeoutLock    sync.Mutex
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

func (sm *StateMachine) AddState(state *State) {
	sm.states[state.ID] = state
}

func (sm *StateMachine) Start(initialState string) bool {
	return sm.internalSwitch(initialState, true)
}

func (sm *StateMachine) runStateEventWithTimeout(event StateEvent, timeout time.Duration, timeoutType TimeoutType) {
	if event == nil {
		return
	}

	ch := make(chan struct{}, 1)
	go func() {
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
			sm.onTimeout(sm, timeoutType)
		}
	}
}

func (sm *StateMachine) leaveState(state *State, triggerEvents bool) {
	if state != nil && triggerEvents && state.OnLeave != nil {
		if state.OnLeaveTimeout.Nanoseconds() > 0 {
			sm.runStateEventWithTimeout(state.OnLeave, state.OnLeaveTimeout, TimeoutLeave)
		} else {
			state.OnLeave(sm)
		}
	}
}

func (sm *StateMachine) enterState(state *State, triggerEvents bool) bool {
	atomic.StorePointer(&sm.current, unsafe.Pointer(state))
	if state != nil {
		if state.OnEnter != nil && triggerEvents {
			if state.OnEnterTimeout.Nanoseconds() > 0 {
				sm.runStateEventWithTimeout(state.OnEnter, state.OnEnterTimeout, TimeoutEnter)
			} else {
				state.OnEnter(sm)
			}
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
						sm.onTimeout(sm, TimeoutState)
					}
				}
			}()
		}

		return true
	}

	return false
}

func (sm *StateMachine) internalSwitch(toState string, triggerEvents bool) bool {
	sm.timeoutLock.Lock()
	if sm.timeoutTracker != nil {
		close(sm.timeoutTracker)
		sm.timeoutTracker = nil
	}
	sm.timeoutLock.Unlock()

	sm.leaveState(sm.CurrentState(), triggerEvents)

	if toState != NoState {
		nState, _ := sm.states[toState]

		return sm.enterState(nState, triggerEvents)
	}

	return false
}

func (sm *StateMachine) Advance(async ...bool) bool {
	sm.advanceLock.Lock()
	defer sm.advanceLock.Unlock()

	if sm.current == nil {
		return false
	}

	advanceId := NoState
	current := sm.CurrentState()

	if current != nil && current.Selector != nil {
		advanceId = current.Selector(current)
	}

	return sm.internalSwitch(advanceId, true)
}

func (sm *StateMachine) CurrentState() *State {
	ptr := atomic.LoadPointer(&sm.current)
	if ptr == nil {
		return nil
	}

	return (*State)(ptr)
}

func (sm *StateMachine) EmergencySwitch(stateId string, triggerEvents ...bool) bool {
	sm.advanceLock.Lock()
	defer sm.advanceLock.Unlock()

	return sm.internalSwitch(stateId, len(triggerEvents) > 0 && triggerEvents[0])
}
