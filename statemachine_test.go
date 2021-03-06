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

	if res, _ := sm.Start("01", true); res {
		action := "process current state: " + sm.CurrentState().ID
		actions = append(actions, action)
	}

	for {
		res, _ := sm.Advance()
		if !res {
			break
		}
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

	timeoutTypeStr := func(tt EventType) string {
		switch tt {
		case EventEnter:
			return "enter"
		case EventLeave:
			return "leave"
		case EventState:
			return "state"
		}

		return ""
	}

	sm := NewStateMachine()
	sm.WithTimeoutHandler(func(sm *StateMachine, timeoutType EventType) {
		aLock.Lock()
		actions = append(actions, "timeout for "+sm.CurrentState().ID+" on type "+timeoutTypeStr(timeoutType))
		aLock.Unlock()
	})

	onEnter := func(sm *StateMachine) {
		action := "enter " + sm.CurrentState().ID
		aLock.Lock()
		actions = append(actions, action)
		aLock.Unlock()
		if sm.CurrentState().ID == "01" {
			time.Sleep(time.Second * 2)
		}
	}

	onLeave := func(sm *StateMachine) {
		action := "leave " + sm.CurrentState().ID
		aLock.Lock()
		actions = append(actions, action)
		aLock.Unlock()
		if sm.CurrentState().ID == "01" {
			time.Sleep(time.Second * 2)
		}
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
		ID:           "02",
		OnEnter:      onEnter,
		OnLeave:      onLeave,
		StateTimeout: time.Second,
		Selector: func(state *State) string {
			return "03"
		},
	})
	sm.AddState(&State{
		ID:      "03",
		OnEnter: onEnter,
		OnLeave: onLeave,
	})

	// trigger enter timeout
	sm.Start("01", true)
	// trigger being at state timeout
	time.Sleep(time.Second * 2)
	// trigger leave timeout
	sm.Advance()
	// trigger normal transfer with timeout cancellation
	sm.Advance()

	expected := []string{
		"enter 01",
		"timeout for 01 on type enter",
		"timeout for 01 on type state",
		"leave 01",
		"timeout for 01 on type leave",
		"enter 02",
		"leave 02",
		"enter 03",
	}

	verifyActions(t, expected, actions)
}

func TestStateMachineAutoAdvance(t *testing.T) {
	sm := NewStateMachine()

	app := &TestApp{
		desiredSequence: []string{"01", "02", "03", "05", "08", "09", "11"},
	}

	actions := []string{}

	onEnter := func(sm *StateMachine) {
		action := "enter and process " + sm.CurrentState().ID
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

	sm.Start("01", true)
	sm.AutoAdvance(time.Second, []string{"05"})

	expected := []string{
		"enter and process 01",
		"leave 01",
		"enter and process 02",
		"leave 02",
		"enter and process 03",
		"leave 03",
		"enter and process 05",
	}

	verifyActions(t, expected, actions)
}

func TestStateMachineErrorHandling(t *testing.T) {
	actions := []string{}

	sm := NewStateMachine()
	sm.WithErrorHandler(func(err interface{}, eventType EventType) {
		errS, _ := err.(string)
		actions = append(actions, errS)
	})

	onEnter := func(sm *StateMachine) {
		if sm.CurrentState().ID == "02" {
			panic("explicit panic")
		}

		action := "enter " + sm.CurrentState().ID
		actions = append(actions, action)
	}

	onLeave := func(sm *StateMachine) {
		action := "leave " + sm.CurrentState().ID
		actions = append(actions, action)
	}

	sm.AddState(&State{
		ID:      "01",
		OnEnter: onEnter,
		OnLeave: onLeave,
		Selector: func(state *State) string {
			return "02"
		},
	})
	sm.AddState(&State{
		ID:      "02",
		OnEnter: onEnter,
		OnLeave: onLeave,
		Selector: func(state *State) string {
			return NoState
		},
	})

	sm.Start("01", true)
	sm.Advance()

	expected := []string{
		"enter 01",
		"leave 01",
		"explicit panic",
	}

	verifyActions(t, expected, actions)
}

func TestStateMachineAutoAdvanceErrorHandling(t *testing.T) {
	actions := []string{}
	state2counter := 0

	sm := NewStateMachine()
	sm.WithErrorHandler(func(err interface{}, eventType EventType) {
		errS, _ := err.(string)
		actions = append(actions, errS)
	})

	app := &TestApp{
		desiredSequence: []string{"01", "02", "03", "05", "08", "09", "11"},
	}

	onEnter := func(sm *StateMachine) {
		st := sm.CurrentState().ID
		if st == "03" {
			panic("explicit panic at 03")
		}

		action := "enter and process " + sm.CurrentState().ID
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
			ID:      stateId,
			OnEnter: onEnter,
			OnLeave: onLeave,
			Selector: func(state *State) string {
				ns := NoState
				for i := 0; i < len(app.desiredSequence)-1; i++ {
					if app.desiredSequence[i] == state.ID {
						ns = app.desiredSequence[i+1]
					}
				}

				if ns == "03" {
					if state2counter < 2 {
						state2counter++
						action := "pending in 02"
						actions = append(actions, action)

						return NoState
					}
				}

				return ns
			},
		})
	}

	sm.Start("01", true)
	sm.AutoAdvance(time.Second, []string{"09"})

	expected := []string{
		"enter and process 01",
		"leave 01",
		"enter and process 02",
		"pending in 02",
		"pending in 02",
		"leave 02",
		"explicit panic at 03",
	}

	verifyActions(t, expected, actions)
}
