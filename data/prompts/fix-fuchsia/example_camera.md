# Examples: fixing nodes from fuchsia.camera.syz.txt

Some Fuchsia FIDL methods reference structs from another library. When `fidlgen_syzkaller` emits a camera node that refers to `fuchsia_sysmem_BufferCollectionInfoInLine`, `fuchsia_sysmem_BufferCollectionInfoOutOfLine`, or `fuchsia_sysmem_BufferCollectionInfoHandles`, `syz-extract` may report them as unknown if the generated spec file does not also contain the external sysmem nodes. Do not fix this by deleting the offending fields. Those fields are part of the FIDL wire payload and removing them changes what syzkaller can generate.

For `unknown type`, first query the FIDL IR to confirm the field is real, then add the missing syzlang representation of the referenced type and any dependent types needed by that representation.

---

## Example 1: an external inline/out-of-line struct (`BufferCollectionInfo`)

### Input

```syzlang
fuchsia_camera_ControlCreateStreamRequest {
    hdr                         fidl_message_header[4633303425909463570]
    buffer_collectionInLine     fuchsia_sysmem_BufferCollectionInfoInLine
    rateInLine                  fuchsia_camera_FrameRateInLine
    stream                      flags[fidl_handle_presence, int32]
    stream_token                flags[fidl_handle_presence, int32]
    buffer_collectionOutOfLine  fuchsia_sysmem_BufferCollectionInfoOutOfLine
    rateOutOfLine               fuchsia_camera_FrameRateOutOfLine
} [packed]
```

```
28:27: unknown type fuchsia_sysmem_BufferCollectionInfoInLine
32:29: unknown type fuchsia_sysmem_BufferCollectionInfoOutOfLine
```

### Thought

The failing fields are `buffer_collectionInLine` and `buffer_collectionOutOfLine`. I should not remove them. I look up the request payload in the FIDL IR.

(call `fidl_get_declaration("fuchsia_camera_ControlCreateStreamRequest")`)

```
// kind: struct
struct fuchsia.camera/ControlCreateStreamRequest {
    buffer_collection fuchsia.sysmem/BufferCollectionInfo
    rate fuchsia.camera/FrameRate
    stream server_end:fuchsia.camera/Stream
    stream_token handle<eventpair>
}
```

The IR confirms that `buffer_collection` is required and has type `fuchsia.sysmem/BufferCollectionInfo`. I query that declaration and recursively query the structs it uses (`BufferFormat`, `ImageFormat`, `PixelFormat`, `ColorSpace`, `ImagePlane`, and `FormatModifier`) to recover the fixed-size wire layout.

(call `fidl_get_declaration("fuchsia_sysmem_BufferCollectionInfo")`)

```
// kind: struct
struct fuchsia.sysmem/BufferCollectionInfo {
    buffer_count uint32
    format fuchsia.sysmem/BufferFormat
    vmos array<handle<vmo>?, 64>
    vmo_size uint64
}
```

`BufferCollectionInfo` is an inline struct. Its `vmos` handles are represented inline as 64 FIDL handle-presence slots and in the handle table as `array[zx_vmo, 64]`. Its `format` field contributes nested inline/out-of-line structs. The request node itself is already correct; the repair is to add the missing sysmem nodes that the request references.

When reconstructing each `...InLine` node, read the wire annotations that `fidl_get_declaration` prints: the per-field `pad=P` tags tell me exactly where to insert `padding array[const[0, int8], P]` fillers (e.g. `buffer_count` at offset 0 is 4 bytes and is followed by 4 padding bytes before `format` at offset 8), the `out_of_line<=..` tags tell me which fields need an `...OutOfLine` entry, and the declaration's `wire_v2: max_out_of_line=0`/`max_handles=0` tells me when an `...OutOfLine`/`...Handles` node collapses to `void void`. I derive the padding and layout from these annotations rather than counting bytes by hand.

### Final output

```syzlang
fuchsia_camera_ControlCreateStreamRequest {
    hdr                         fidl_message_header[4633303425909463570]
    buffer_collectionInLine     fuchsia_sysmem_BufferCollectionInfoInLine
    rateInLine                  fuchsia_camera_FrameRateInLine
    stream                      flags[fidl_handle_presence, int32]
    stream_token                flags[fidl_handle_presence, int32]
    buffer_collectionOutOfLine  fuchsia_sysmem_BufferCollectionInfoOutOfLine
    rateOutOfLine               fuchsia_camera_FrameRateOutOfLine
} [packed]

fuchsia_sysmem_ImagePlaneInLine {
    byte_offset     int32
    bytes_per_row   int32
} [packed]

fuchsia_sysmem_ImagePlaneOutOfLine {
    void    void
} [packed]

fuchsia_sysmem_ImagePlaneHandles {
    void    void
} [packed]

fuchsia_sysmem_ColorSpaceInLine {
    type    int32
} [packed]

fuchsia_sysmem_ColorSpaceOutOfLine {
    void    void
} [packed]

fuchsia_sysmem_ColorSpaceHandles {
    void    void
} [packed]

fuchsia_sysmem_FormatModifierInLine {
    value   int64
} [packed]

fuchsia_sysmem_FormatModifierOutOfLine {
    void    void
} [packed]

fuchsia_sysmem_FormatModifierHandles {
    void    void
} [packed]

fuchsia_sysmem_PixelFormatInLine {
    type                    int32
    has_format_modifier     int8
    padding0                array[const[0, int8], 3]
    format_modifierInLine   fuchsia_sysmem_FormatModifierInLine
} [packed]

fuchsia_sysmem_PixelFormatOutOfLine {
    format_modifierOutOfLine    fuchsia_sysmem_FormatModifierOutOfLine
} [packed]

fuchsia_sysmem_PixelFormatHandles {
    format_modifier fuchsia_sysmem_FormatModifierHandles
} [packed]

fuchsia_sysmem_ImageFormatInLine {
    width                   int32
    height                  int32
    layers                  int32
    padding0                array[const[0, int8], 4]
    pixel_formatInLine      fuchsia_sysmem_PixelFormatInLine
    color_spaceInLine       fuchsia_sysmem_ColorSpaceInLine
    planesInLine            array[fuchsia_sysmem_ImagePlaneInLine, 4]
    padding1                array[const[0, int8], 4]
} [packed]

fuchsia_sysmem_ImageFormatOutOfLine {
    pixel_formatOutOfLine   fuchsia_sysmem_PixelFormatOutOfLine
    color_spaceOutOfLine    fuchsia_sysmem_ColorSpaceOutOfLine
    planesOutOfLine         array[fuchsia_sysmem_ImagePlaneOutOfLine, 4]
} [packed]

fuchsia_sysmem_ImageFormatHandles {
    pixel_format    fuchsia_sysmem_PixelFormatHandles
    color_space     fuchsia_sysmem_ColorSpaceHandles
    planes          array[fuchsia_sysmem_ImagePlaneHandles, 4]
} [packed]

fuchsia_sysmem_BufferFormatInLine {
    tag             int32
    padding0        array[const[0, int8], 4]
    imageInLine     fuchsia_sysmem_ImageFormatInLine
} [packed]

fuchsia_sysmem_BufferFormatOutOfLine {
    imageOutOfLine  fuchsia_sysmem_ImageFormatOutOfLine
} [packed]

fuchsia_sysmem_BufferFormatHandles {
    image   fuchsia_sysmem_ImageFormatHandles
} [packed]

fuchsia_sysmem_BufferCollectionInfoInLine {
    buffer_count    int32
    padding0        array[const[0, int8], 4]
    formatInLine    fuchsia_sysmem_BufferFormatInLine
    vmos            array[flags[fidl_handle_presence, int32], 64]
    vmo_size        int64
} [packed]

fuchsia_sysmem_BufferCollectionInfoOutOfLine {
    formatOutOfLine fuchsia_sysmem_BufferFormatOutOfLine
} [packed]
```

---

## Example 2: a missing handle table and a missing handle resource

### Input

```syzlang
fuchsia_camera_ControlCreateStreamRequestHandles {
    buffer_collection   fuchsia_sysmem_BufferCollectionInfoHandles
    rate                fuchsia_camera_FrameRateHandles
    stream              zx_chan_fuchsia_camera_Stream_server
    stream_token        zx_eventpair
} [packed]
```

```
36:20: unknown type fuchsia_sysmem_BufferCollectionInfoHandles
39:16: unknown type zx_eventpair
```

### Thought

The request's handle table has two missing references. I query the same FIDL request payload to confirm both handle-bearing fields.

(call `fidl_get_declaration("fuchsia_camera_ControlCreateStreamRequestHandles")`)

```
// kind: struct
struct fuchsia.camera/ControlCreateStreamRequest {
    buffer_collection fuchsia.sysmem/BufferCollectionInfo
    rate fuchsia.camera/FrameRate
    stream server_end:fuchsia.camera/Stream
    stream_token handle<eventpair>
}
```

`buffer_collection` is still `fuchsia.sysmem/BufferCollectionInfo`, whose FIDL definition contains `vmos array<handle<vmo>?, 64>`. Therefore the handles node must include a nested `BufferFormatHandles` field and the 64 VMO handles. The `stream_token` field is FIDL `handle<eventpair>`, so the syzlang resource should be `zx_eventpair` derived from `zx_handle`, not a plain integer and not a deleted field.

### Final output

```syzlang
resource zx_eventpair[zx_handle]

fuchsia_camera_ControlCreateStreamRequestHandles {
    buffer_collection   fuchsia_sysmem_BufferCollectionInfoHandles
    rate                fuchsia_camera_FrameRateHandles
    stream              zx_chan_fuchsia_camera_Stream_server
    stream_token        zx_eventpair
} [packed]

fuchsia_sysmem_ImagePlaneHandles {
    void    void
} [packed]

fuchsia_sysmem_ColorSpaceHandles {
    void    void
} [packed]

fuchsia_sysmem_FormatModifierHandles {
    void    void
} [packed]

fuchsia_sysmem_PixelFormatHandles {
    format_modifier fuchsia_sysmem_FormatModifierHandles
} [packed]

fuchsia_sysmem_ImageFormatHandles {
    pixel_format    fuchsia_sysmem_PixelFormatHandles
    color_space     fuchsia_sysmem_ColorSpaceHandles
    planes          array[fuchsia_sysmem_ImagePlaneHandles, 4]
} [packed]

fuchsia_sysmem_BufferFormatHandles {
    image   fuchsia_sysmem_ImageFormatHandles
} [packed]

fuchsia_sysmem_BufferCollectionInfoHandles {
    format  fuchsia_sysmem_BufferFormatHandles
    vmos    array[zx_vmo, 64]
} [packed]
```

---

## Takeaways

- For `unknown type` on a generated `...InLine`, `...OutOfLine`, or `...Handles` node, do not delete the field. Query the FIDL IR and add the missing referenced node if the field exists in the original FIDL declaration.
- External library types keep the same mangling rule: `fuchsia.sysmem/BufferCollectionInfo` becomes `fuchsia_sysmem_BufferCollectionInfoInLine`, `fuchsia_sysmem_BufferCollectionInfoOutOfLine`, and `fuchsia_sysmem_BufferCollectionInfoHandles`.
- Handles are split across the inline payload and handle table. A FIDL `array<handle<vmo>?, 64>` becomes 64 inline handle-presence slots plus `array[zx_vmo, 64]` in the handles node.
- A FIDL `handle<eventpair>` should be modeled as a `zx_eventpair` resource derived from `zx_handle` if that resource is missing.
- The fix may need to include dependent structs recursively, but each dependency should still be justified by the FIDL IR. Do not invent fields or collapse a struct to `void` unless the IR says it has no data or handles.
- Let the wire annotations do the arithmetic: `pad=P` gives the exact `padding array[const[0, int8], P]` filler, `out_of_line<=..`/`handles<=..` on a field decide whether it appears in the `...OutOfLine`/`...Handles` node, and the declaration's `wire_v2: max_out_of_line`/`max_handles` decide whether those nodes are `void void`.
