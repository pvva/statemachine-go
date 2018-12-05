This repository contains simple state machine implementation in Go.

# State machine

State machine enables you to define states, distinguished by string identifier, transitions between states, on enter and on leave events with timeouts per each event, as well as timeout for being at state along with global timeout handler.

## Defining state machine
```
    sm := statemachine.NewStateMachine()
    // optional error handler
    sm.WithErrorHandler(func(err interface{}, eventType EventType) {
        ...
    })

    sm.AddState(&statemachine.State{
        ID: "01",
        OnEnter: onEnter,
        OnLeave: onLeave,
        Selector: selector,
    }
    sm.AddState(&statemachine.State{
        ID: "02",
        OnEnter: onEnter,
        OnLeave: onLeave,
		OnEnterTimeout: time.Second,
		OnLeaveTimeout: time.Second,
		StateTimeout:   time.Second * 10,
        Selector: selector,
    }
    ...
```

```
    sm := statemachine.NewStateMachine(func(sm *statemachine.StateMachine, eventType statemachine.EventType) {
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

    if res, err := sm.Advance(); res && err != nil {
        // state machine has advanced to next state
    } else {
        // state machine has stopped or errored
    }
```

## Emergency switch to chosen state
```
    // by default such switch doesn't trigger events
    res, err := sm.EmergencySwitch("03")

    // energency switch with events triggering
    res, err := sm.EmergencySwitch("03", true)
```

## Make machine advance automatically
```
    // try to advance each second
    sm.AutoAdvance(time.Second, []string{"desired_terminal_state1", "desired_terminal_state2"})
```

For complete example see test.
