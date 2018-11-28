package statemachine

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

type TestApp struct {
	desiredSequence []string
}

func (ta *TestApp) SelectNextState(state *State) string {
	for i := 0; i < len(ta.desiredSequence)-1; i++ {
		if ta.desiredSequence[i] == state.ID {
			return ta.desiredSequence[i+1]
		}
	}

	return NoState
}

func verifyActions(t *testing.T, expected []string, actions []string) {
	if len(actions) != len(expected) {
		t.Fatal("actual actions differ from expected")
	}

	for i := 0; i < len(actions); i++ {
		if actions[i] != expected[i] {
			t.Fatal("actual action differs from expected one: ", actions[i], " <> ", expected[i])
		}
	}
}

func TestStateMachine(t *testing.T) {
	sm := NewStateMachine()

	app := &TestApp{
		desiredSequence: []string{"01", "02", "03", "05", "08", "09", "11"},
	}

	actions := []string{}

	onEnter := func(sm *StateMachine) {
		action := "enter " + sm.CurrentState().ID
		actions = append(actions, action)
	}

	onLeave := func(sm *StateMachine) {
		action := "leave " + sm.CurrentState().ID
		actions = append(actions, action)
	}

	for i := 1; i <= 10; i++ {
		stateId := strconv.Itoa(i)
		if i < 10 {
			stateId = "0" + stateId
		}
		sm.AddState(&State{
			ID:       stateId,
			OnEnter:  onEnter,
			OnLeave:  onLeave,
			Selector: app.SelectNextState,
		})
	}

	if sm.Start("01") {
		action := "process current state: " + sm.CurrentState().ID
		actions = append(actions, action)
	}

	for sm.Advance() {
		action := "process current state: " + sm.CurrentState().ID
		actions = append(actions, action)
	}

	expected := []string{
		"enter 01",
		"process current state: 01",
		"leave 01",
		"enter 02",
		"process current state: 02",
		"leave 02",
		"enter 03",
		"process current state: 03",
		"leave 03",
		"enter 05",
		"process current state: 05",
		"leave 05",
		"enter 08",
		"process current state: 08",
		"leave 08",
		"enter 09",
		"process current state: 09",
		"leave 09",
	}

	verifyActions(t, expected, actions)

	sm.EmergencySwitch("03")
	if sm.CurrentState().ID != "03" {
		t.Fatal("emergency state switch failed")
	}
}

func TestStateMachineTimeouts(t *testing.T) {
	aLock := sync.Mutex{}
	actions := []string{}

	timeoutTypeStr := func(tt TimeoutType) string {
		switch tt {
		case TimeoutEnter:
			return "enter"
		case TimeoutLeave:
			return "leave"
		case TimeoutState:
			return "state"
		}

		return ""
	}

	sm := NewStateMachine(func(sm *StateMachine, timeoutType TimeoutType) {
		aLock.Lock()
		actions = append(actions, "timeout for "+sm.CurrentState().ID+" on type "+timeoutTypeStr(timeoutType))
		aLock.Unlock()
	})

	onEnter := func(sm *StateMachine) {
		action := "enter " + sm.CurrentState().ID
		aLock.Lock()
		actions = append(actions, action)
		aLock.Unlock()
		time.Sleep(time.Second * 2)
	}

	onLeave := func(sm *StateMachine) {
		action := "leave " + sm.CurrentState().ID
		aLock.Lock()
		actions = append(actions, action)
		aLock.Unlock()
		time.Sleep(time.Second * 2)
	}

	sm.AddState(&State{
		ID:             "01",
		OnEnter:        onEnter,
		OnLeave:        onLeave,
		OnEnterTimeout: time.Second,
		OnLeaveTimeout: time.Second,
		StateTimeout:   time.Second,
		Selector: func(state *State) string {
			return "02"
		},
	})
	sm.AddState(&State{
		ID:      "02",
		OnEnter: onEnter,
		OnLeave: onLeave,
	})

	// trigger enter timeout
	sm.Start("01")
	// trigger being at state timeout
	time.Sleep(time.Second * 2)
	// trigger leave timeout
	sm.Advance()

	expected := []string{
		"enter 01",
		"timeout for 01 on type enter",
		"timeout for 01 on type state",
		"leave 01",
		"timeout for 01 on type leave",
		"enter 02",
	}

	verifyActions(t, expected, actions)
}
