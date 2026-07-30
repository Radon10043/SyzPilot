# Example: outlining todo tasks for the RTC driver protocol

## Input

```json
[
  {
    "kind": "twoway",
    "ordinal": 2881649623829671924,
    "name": "Get",
    "has_request": true,
    "maybe_attributes": [
      {"name": "doc", "arguments": [{"name": "value", "type": "string",
        "value": {"kind": "literal", "value": " Get the current RTC time.\n"}}]}
    ],
    "maybe_request_payload": null,
    "has_response": true,
    "maybe_response_payload": {
      "kind_v2": "identifier",
      "identifier": "fuchsia.hardware.rtc/Device_Get_Result",
      "type_shape_v2": {"inline_size": 16, "alignment": 8, "max_handles": 0, "max_out_of_line": 8}
    }
  },
  {
    "kind": "twoway",
    "ordinal": 1614978035535395557,
    "name": "Set2",
    "has_request": true,
    "maybe_request_payload": {
      "kind_v2": "identifier",
      "identifier": "fuchsia.hardware.rtc/DeviceSet2Request",
      "type_shape_v2": {"inline_size": 8, "alignment": 2, "max_handles": 0, "max_out_of_line": 0}
    },
    "has_response": true,
    "maybe_response_payload": {
      "kind_v2": "identifier",
      "identifier": "fuchsia.hardware.rtc/Device_Set2_Result",
      "type_shape_v2": {"inline_size": 16, "alignment": 8, "max_handles": 0, "max_out_of_line": 0}
    }
  }
]
```

## Output

### Thought

**Identify the protocol**: the payload identifiers are `fuchsia.hardware.rtc/DeviceSet2Request` and
`fuchsia.hardware.rtc/Device_Get_Result`. Library `fuchsia.hardware.rtc`, protocol `Device`, methods
`Get` and `Set2`. Confirming.

(Calling `get_protocol_methods_by_name("fuchsia.hardware.rtc/Device")`)

The tool returns the same two methods, so the protocol FQN is `fuchsia.hardware.rtc/Device` and the
mangled name is `fuchsia_hardware_rtc_Device`.

**Check discoverability**:

(Calling `get_protocol_attrs_by_name("fuchsia.hardware.rtc/Device")`)

```json
[{"name": "discoverable", "arguments": []}]
```

Discoverable, so the service is reachable at `/svc/fuchsia.hardware.rtc.Device`. The generate stage
will use that path in `fdio_service_connect$fuchsia_hardware_rtc_Device`; here it only confirms that
the connect call is worth emitting.

**Classify the methods**: both are `twoway`, so both become `zx_channel_call$...` variants. `Get`
has no request payload — its request message is just the FIDL header, which is still a real message
worth sending, so it gets a task like any other method.

**Initialization**: the client end of the channel comes from `zx_channel_create$...`, and the server
end must be handed to the RTC component with `fdio_service_connect$...`. Both are `init_syscall`
so that they are processed before the calls that consume the client handle.

### Final output

```json
[
	{"type": "init_syscall", "name": "zx_channel_create$fuchsia_hardware_rtc_Device"},
	{"type": "init_syscall", "name": "fdio_service_connect$fuchsia_hardware_rtc_Device"},
	{"type": "syscall", "name": "zx_channel_call$fuchsia_hardware_rtc_DeviceGet"},
	{"type": "syscall", "name": "zx_channel_call$fuchsia_hardware_rtc_DeviceSet2"}
]
```
