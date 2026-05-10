# Decision Log

## Useful emojis = ⌛, ✔️ 
---

## Slice 6: DEC-001 · FSM for protocol change
**Status:** ⌛ In progress

### Context
Once protocol updation is complete HTTP is never used and both the client and server continue with WebSockets. However we do need to send an HTTP request initially. So in order to manage what protocol is being used, I thought of implementing a FINITE STATE MACHINE, having states as HTTP or WebSocket. Whatever the state the client or the goroutine for that client is in, the requests and responses will be handled accordingly.

### Decision
Use a Finite State Machine with states `StateHTTP` and `StateWebSocket`.
Initial state = `StateHTTP`
Transition is triggered by--
1. Server = after sending the 101 response but the sec-websocket-accept key.
2. Client = after validating the received key with computed one.

### Alternatives Considered
| Option | Reason Rejected |
|---|---|
| `Single for loop with mode flag` | Simple but hard to maintain and extend as complexity grows. Difficult to debug since everything is crammed in one place. |
| `Channel-based handoff` | Defeats the purpose of raw implementation. Concurrency and not networking primitives. |

### Consequences
- Structured code and intutive logic flow.
- Deep understanding how protocol upgradation.
- Can even be extend to include other protocols if needed.

---

## Slice 6: DEC-002 · Creating structures for client and server.
**Status:** ⌛ In progress

### Context
Since each client is tackled by different threads/goroutines. Different clients might be at different protocols at any given instance of time. Also, it is equally likely that some clients might never upgrade to a WebSocket protocol and might simply want to continue with HTTP. 
Furthermore there are plethora of parameters related to a client which needs to be kept track of -- TCP connection, Current Protocol State, Keys exchanged, Buffers, Masks etc.
Thus structure provides a better way to keep all the data for a particular client organized at a place.

#### But why do I need it for the server ?
To hold shared server-wide data in one place -- listener, port, list of active clients (needed later for our scope), etc.
In Nutshell = organized code.

### Decision
Created structures for organizing data pertaining to clients and server.

### Alternatives Considered
| Option | Reason Rejected |
|---|---|
|`No structures` | Would have ended up with excess global variables and parameters passed to functions. Messy and bug prone. |

### Consequences
- Better code and easier to debug.
- Analogous to concept of classes and objects in OOP.
- Can be scaled reliably.

---
