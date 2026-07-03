# Examples: fixing nodes from fuchsia.bluetooth.syz.txt

The most common defect in machine-generated Fuchsia syzlang is a **malformed union member**: `fidl_union_member[N, ]` with no field name and an empty type argument. `fidlgen_syzkaller` emits this whenever it cannot represent a union option (FIDL `framework_error`, table payloads, etc.). To fix it you must recover the member's real name and a valid syzlang type from the FIDL definition.

A `unionname [` ... `]` block holds one member per line: `membername fidl_union_member[ordinal, type]`. The ordinal in the broken line tells you which FIDL union member is missing.

---

## Example 1: a flexible method result union (`framework_err`)

### Input

```syzlang
fuchsia_bluetooth_Channel_Send_ResultInLine [
    responseInLine fidl_union_member[1, fuchsia_bluetooth_Channel_Send_ResponseInLine]
     fidl_union_member[3, ]
]
```

```
158:23: unexpected '[', expecting int, identifier, string
159:1: unexpected ']', expecting comment, define, include, resource, identifier
```

### Thought

The member with ordinal `1` (`responseInLine`) is fine. The member with ordinal `3` has no name and an empty type. I look up the FIDL union to find ordinal 3.

(call `fidl_get_declaration("fuchsia_bluetooth_Channel_Send_ResultInLine")`)

```
// kind: union
union fuchsia.bluetooth/Channel_Send_Result {
    1: response fuchsia.bluetooth/Channel_Send_Response
    3: framework_err framework_error
}
```

Ordinal 3 is `framework_err` of FIDL type `framework_error`. `framework_error` is the flexible-interaction transport error: a 32-bit value. So the member name is `framework_err` and the syzlang type is `int32`. I leave ordinal 1 untouched.

### Final output

```syzlang
fuchsia_bluetooth_Channel_Send_ResultInLine [
    responseInLine fidl_union_member[1, fuchsia_bluetooth_Channel_Send_ResponseInLine]
    framework_err fidl_union_member[3, int32]
]
```

---

## Example 2: a domain union with a table option (`vendor_format`)

### Input

```syzlang
fuchsia_bluetooth_CodecIdInLine [
    assigned_format fidl_union_member[1, flags[fuchsia_bluetooth_AssignedCodingFormat, int8]]
     fidl_union_member[2, ]
]
```

```
178:23: unexpected '[', expecting int, identifier, string
179:1: unexpected ']', expecting comment, define, include, resource, identifier
```

### Thought

Ordinal 1 (`assigned_format`) is already correct. Ordinal 2 is empty. I look up the union, then the type of its second member.

(call `fidl_get_declaration("fuchsia_bluetooth_CodecIdInLine")`)

```
// kind: union
union fuchsia.bluetooth/CodecId {
    1: assigned_format fuchsia.bluetooth/AssignedCodingFormat
    2: vendor_format fuchsia.bluetooth/VendorCodingFormat
}
```

Ordinal 2 is `vendor_format` of type `VendorCodingFormat`. Let me see what that is.

(call `fidl_get_declaration("fuchsia_bluetooth_VendorCodingFormat")`)

```
// kind: table
table fuchsia.bluetooth/VendorCodingFormat {
    1: company_id uint16
    2: vendor_id uint16
}
```

`VendorCodingFormat` is a FIDL **table**. In the Fuchsia FIDL wire format a table is encoded inline as an envelope vector, which syzkaller models with the `fidl_vector` helper type. So the member name is `vendor_format` and the inline type is `fidl_vector`.

### Final output

```syzlang
fuchsia_bluetooth_CodecIdInLine [
    assigned_format fidl_union_member[1, flags[fuchsia_bluetooth_AssignedCodingFormat, int8]]
    vendor_format fidl_union_member[2, fidl_vector]
]
```

---

## Takeaways

- Map the empty member's **ordinal** to the FIDL union member with the same ordinal; use that member's name and type.
- Translate the FIDL type to syzlang: a scalar/`framework_error` becomes the matching `intN`; an enum/bits becomes `flags[...]`; a `vector`/`string`/table payload becomes `fidl_vector`/`fidl_string`; a struct option becomes its generated `...InLine` node.
- Change only the broken line(s). Keep correct members and the node name as-is.
