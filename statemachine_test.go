package statemachine

import (
	"testing"
)

type TestApp struct {
	desiredSequence []string
}

func (ta *TestApp) Condition(state State) string {
	for i := 0; i < len(ta.desiredSequence)-1; i++ {
		if ta.desiredSequence[i] == state.ID() {
			return ta.desiredSequence[i+1]
		}
	}

	return NoState
}

func TestStateMachine(t *testing.T) {
	sm := NewStateMachine()

	app := &TestApp{
		desiredSequence: []string{"01", "02", "03", "05", "08", "09", "11"},
	}

	actions := []string{}

	onEnter := func(st State) {
		action := "enter " + st.ID()
		actions = append(actions, action)
	}

	onLeave := func(st State) {
		action := "leave " + st.ID()
		actions = append(actions, action)
	}

	sm.AddState("01", []string{"02"}, onEnter, onLeave, app.Condition)
	sm.AddState("02", []string{"03", "04"}, onEnter, onLeave, app.Condition)
	sm.AddState("03", []string{"05"}, onEnter, onLeave, app.Condition)
	sm.AddState("04", []string{"06", "07"}, onEnter, onLeave, app.Condition)
	sm.AddState("05", []string{"08"}, onEnter, onLeave, app.Condition)
	sm.AddState("06", []string{"08"}, onEnter, onLeave, app.Condition)
	sm.AddState("07", []string{"10"}, onEnter, onLeave, app.Condition)
	sm.AddState("08", []string{"09"}, onEnter, onLeave, app.Condition)
	sm.AddState("09", []string{"11"}, onEnter, onLeave, app.Condition)
	sm.AddState("10", []string{"11"}, onEnter, onLeave, app.Condition)

	sm.Bind()
	for sm.Advance() {
		action := "process current state: " + sm.CurrentState().ID()
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

	if len(actions) != len(expected) {
		t.Fatal("actual actions differ from expected")
	}

	for i := 0; i < len(actions); i++ {
		if actions[i] != expected[i] {
			t.Fatal("actual action differs from expected one: ", actions[i], " <> ", expected[i])
		}
	}

	sm.EmergencySwitch("03")
	if sm.CurrentState().ID() != "03" {
		t.Fatal("emergency state switch failed")
	}
}
