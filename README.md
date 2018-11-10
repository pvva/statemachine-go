This repository contains simple state machine implementation in Go.

# State machine

State machine enables you to define states, distinguished by string identifier, transitions between states, on enter and on leave events and conditions for certain transitions.

## Defining state machine
```
    sm := statemachine.NewStateMachine()

    sm.AddState("01", []string{"02"}, onEnter, onLeave, condition)
    sm.AddState("02", []string{"03", "04"}, onEnter, onLeave, condition)
    ...
```

## Defining condition function
```
    condition := func(st statemachine.State) string {
        if st.ID() == "02" {
            return "03"
        }

        return statemachine.NoState
    }
```

## Defining enter and leave events
```
    onEnter := func(st statemachine.State) {
        // do something here
    }
    onLeave := func(st statemachine.State) {
        // do something here
    }
```

## Get current state of machine
```
    var state statemachine.State

    ...

    state = sm.CurrentState()
```

## Activating and using state machine
```
    sm.Bind()

    ...

    if sm.Advance() {
        // state machine has advanced to next state
    } else {
        // state machine has stopped
    }
```

## Emergency switch to chosen state
```
    // by default such switch doesn't trigger events
    sm.EmergencySwitch("03")

    // energency switch with events triggering
    sm.EmergencySwitch("03", true)
```

For complete example see test.
