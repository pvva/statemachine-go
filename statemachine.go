package statemachine

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const NoState = ""

type State interface {
	ID() string
}

type StateEvent func(state State)
type StateCondition func(state State) string

type innerState struct {
	id        string
	next      map[string]*innerState
	onEnter   StateEvent
	onLeave   StateEvent
	condition StateCondition
}

func (is *innerState) ID() string {
	return is.id
}

type StateMachine struct {
	root        *innerState
	current     unsafe.Pointer
	states      map[string]*innerState
	advanceLock sync.Mutex
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		states: make(map[string]*innerState),
	}
}

func (sm *StateMachine) AddState(id string, next []string, onEnter, onLeave StateEvent, condition StateCondition) {
	nState := &innerState{
		id:        id,
		next:      make(map[string]*innerState),
		onEnter:   onEnter,
		onLeave:   onLeave,
		condition: condition,
	}
	for _, stateId := range next {
		nState.next[stateId] = nil
	}

	sm.states[id] = nState
	if sm.root == nil {
		sm.root = nState
	}
}

func (sm *StateMachine) Bind() {
	if len(sm.states) == 0 {
		panic("State machine is empty")
	}

	for _, stt := range sm.states {
		for stateId := range stt.next {
			stt.next[stateId], _ = sm.states[stateId]
		}
	}
	for stateId := range sm.root.next {
		sm.root.next[stateId], _ = sm.states[stateId]
	}
}

func (sm *StateMachine) start() {
	if sm.root.onEnter != nil {
		sm.root.onEnter(sm.root)
	}

	atomic.StorePointer(&sm.current, unsafe.Pointer(sm.root))
}

func (sm *StateMachine) internalSwitch(toState string, triggerEvents bool) bool {
	current, notNil := sm.CurrentState().(*innerState)
	if notNil && triggerEvents && current.onLeave != nil {
		current.onLeave(current)
	}
	if toState != NoState {
		nState, _ := sm.states[toState]
		atomic.StorePointer(&sm.current, unsafe.Pointer(nState))
		if nState != nil && nState.onEnter != nil {
			if triggerEvents {
				nState.onEnter(nState)
			}

			return true
		}
	}

	return false
}

func (sm *StateMachine) Advance() bool {
	sm.advanceLock.Lock()
	defer sm.advanceLock.Unlock()

	if sm.current == nil {
		sm.start()

		return sm.current != nil
	}

	current := sm.CurrentState().(*innerState)

	advanceId := NoState
	for nState := range current.next {
		advanceId = nState
		break
	}
	if current.condition != nil {
		advanceId = current.condition(current)
	}

	return sm.internalSwitch(advanceId, true)
}

func (sm *StateMachine) CurrentState() State {
	ptr := atomic.LoadPointer(&sm.current)
	if ptr == nil {
		return nil
	}

	return (*innerState)(ptr)
}

func (sm *StateMachine) EmergencySwitch(stateId string, triggerEvents ...bool) bool {
	sm.advanceLock.Lock()
	defer sm.advanceLock.Unlock()

	return sm.internalSwitch(stateId, len(triggerEvents) > 0 && triggerEvents[0])
}
