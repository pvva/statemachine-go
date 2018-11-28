This repository contains simple state machine implementation in Go.

# State machine

State machine enables you to define states, distinguished by string identifier, transitions between states, on enter and on leave events with timeouts per each event, as well as timeout for being at state along with global timeout handler.

## Defining state machine
```
    sm := statemachine.NewStateMachine()

    sm.AddState(&statemachine.State{
        ID: "01",
        OnEnter: onEnter,
        OnLeave: onLeave,
        Selector: selector,
    }
    ...
```

```
    sm := statemachine.NewStateMachine(func(sm *statemachine.StateMachine, timeoutType statemachine.TimeoutType) {
        // handle timeout
    })

    ...
```

## Defining selector function
```
    selector := func(st *statemachine.State) string {
        if st.ID == "02" {
            return "03"
        }

        return statemachine.NoState
    }
```

## Defining enter and leave events
```
    onEnter := func(sm *statemachine.StateMachine) {
        // do something here
        currentState := sm.CurrentState()
        ...
    }
    onLeave := func(sm *statemachine.StateMachine) {
        // do something here
    }
```

## Get current state of machine
```
    state := sm.CurrentState()
```

## Activating and using state machine
```
    sm.Start("01")

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
