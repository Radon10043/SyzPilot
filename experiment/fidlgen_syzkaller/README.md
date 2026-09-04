# fidlgen_syzkaller

Files under `src` are vendored from `$FUCHSIA/tools/fidl/fidlgen_syzkaller` together with the FIDL IR library it needs (`$FUCHSIA/tools/fidl/lib/fidlgen`, copied under `src/internal/fidlgen`). The commit of `$FUCHSIA/integration` is 4c117a94d1cafc3f48d0f3deee07676f6501a0af. You can build it with `make fidlgen_syzkaller` from the repository root, and the binary lands in `$CLOUD/bin/`.

fidlgen_syzkaller can generates syzkaller syscall specifications from the JSON IR of a FIDL library. The original version has some issues and we have tried our best to fix them.

You can also run `$CLOUD/scripts/fuchsia/fidlgen` to generate syscall specifications in batch.
