# Example: outlining todo tasks for a protocol with oneway methods and events

## Input

```json
[
  {
    "kind": "twoway",
    "ordinal": 8048669970892768790,
    "name": "Read",
    "has_request": true,
    "maybe_request_payload": null,
    "has_response": true,
    "maybe_response_payload": {
      "kind_v2": "identifier",
      "identifier": "fuchsia.bluetooth.bredr/ScoConnection_Read_Result"
    }
  },
  {
    "kind": "twoway",
    "ordinal": 4129267801182741929,
    "name": "Write",
    "has_request": true,
    "maybe_request_payload": {
      "kind_v2": "identifier",
      "identifier": "fuchsia.bluetooth.bredr/ScoConnectionWriteRequest",
      "type_shape_v2": {"inline_size": 16, "alignment": 8, "max_handles": 0}
    },
    "has_response": true,
    "maybe_response_payload": {
      "kind_v2": "identifier",
      "identifier": "fuchsia.bluetooth.bredr/ScoConnection_Write_Result"
    }
  },
  {
    "kind": "event",
    "ordinal": 1817941751101012045,
    "name": "OnConnectionComplete",
    "has_request": false,
    "maybe_request_payload": null,
    "has_response": true,
    "maybe_response_payload": {
      "kind_v2": "identifier",
      "identifier": "fuchsia.bluetooth.bredr/ScoConnectionOnConnectionCompleteRequest"
    }
  },
  {
    "kind": "oneway",
    "ordinal": 1951769149418924631,
    "name": "RequestDisconnect",
    "has_request": true,
    "maybe_request_payload": null,
    "has_response": false,
    "maybe_response_payload": null
  }
]
```

## Output

### Thought

**Identify the protocol**: identifiers are `fuchsia.bluetooth.bredr/ScoConnection*`, so the protocol
is `fuchsia.bluetooth.bredr/ScoConnection`, mangled `fuchsia_bluetooth_bredr_ScoConnection`.

(Calling `get_protocol_methods_by_name("fuchsia.bluetooth.bredr/ScoConnection")`) — confirmed, four
methods.

**Check discoverability**:

(Calling `get_protocol_attrs_by_name("fuchsia.bluetooth.bredr/ScoConnection")`)

```json
[{"name": "doc", "arguments": [{"name": "value", "type": "string"}]}]
```

No `discoverable` attribute. This protocol is normally obtained by being passed over another
channel, not by name from `/svc`. The fuzzer still creates the channel pair and still attempts
`fdio_service_connect` with the bare `/svc/` prefix — the connect may fail at runtime, but the
client handle produced by `zx_channel_create$...` remains usable for writing messages, which is
what the method calls need. Both init syscalls are emitted.

**Classify the four methods**:

- `Read` — `twoway`, no request payload -> `zx_channel_call$fuchsia_bluetooth_bredr_ScoConnectionRead`.
- `Write` — `twoway` with a payload -> `zx_channel_call$fuchsia_bluetooth_bredr_ScoConnectionWrite`.
- `OnConnectionComplete` — `event`, `has_request: false`. The server sends it, so there is nothing
  the fuzzer can *call*; but it can write a message carrying that ordinal onto the client end, which
  exercises the server's unexpected-message handling. Emit it as a `zx_channel_write$...` task; its
  body will be built from `maybe_response_payload`.
- `RequestDisconnect` — `oneway`, no payload -> `zx_channel_write$fuchsia_bluetooth_bredr_ScoConnectionRequestDisconnect`.

Note `ScoConnectionWriteRequest` is a `table`, not a struct. That matters for the layout written in
the generate stage, not for the task list here.

### Final output

```json
[
	{"type": "init_syscall", "name": "zx_channel_create$fuchsia_bluetooth_bredr_ScoConnection"},
	{"type": "init_syscall", "name": "fdio_service_connect$fuchsia_bluetooth_bredr_ScoConnection"},
	{"type": "syscall", "name": "zx_channel_call$fuchsia_bluetooth_bredr_ScoConnectionRead"},
	{"type": "syscall", "name": "zx_channel_call$fuchsia_bluetooth_bredr_ScoConnectionWrite"},
	{"type": "syscall", "name": "zx_channel_write$fuchsia_bluetooth_bredr_ScoConnectionOnConnectionComplete"},
	{"type": "syscall", "name": "zx_channel_write$fuchsia_bluetooth_bredr_ScoConnectionRequestDisconnect"}
]
```
