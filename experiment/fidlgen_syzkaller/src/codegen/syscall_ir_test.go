// Copyright 2025 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package codegen

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/Radon10043/cloud/experiment/fidlgen_syzkaller/src/internal/fidlgen"
)

// The test data is embedded rather than read from disk: the test binary does
// not necessarily run from its source directory.

// testIR is the JSON IR of a handful of representative vDSO protocols, trimmed
// out of the real library zx.
//
//go:embed testdata/syscalls.fidl.json
var testIR []byte

//go:embed testdata/syscalls.syz.txt.golden
var testGolden string

const testGoldenPath = "tools/fidl/fidlgen_syzkaller/codegen/testdata/syscalls.syz.txt.golden"

func readTestIR(t *testing.T) fidlgen.Root {
	t.Helper()
	root, err := fidlgen.ReadJSONIrContent(testIR)
	if err != nil {
		t.Fatalf("failed to decode the test IR: %v", err)
	}
	return root
}

func defaultOptions() SyscallOptions {
	return SyscallOptions{IncludeNext: true, InferHandleSubtypes: true}
}

func TestIsSyscallLibrary(t *testing.T) {
	root := readTestIR(t)
	if !IsSyscallLibrary(root) {
		t.Error("a library of protocols over the Syscall transport was not recognized")
	}
	// A library with no protocols at all is not a syscall library.
	if IsSyscallLibrary(fidlgen.Root{}) {
		t.Error("an empty library was mistaken for a syscall library")
	}
	// Nor is one that merely contains a Syscall protocol: test libraries
	// declare one alongside ordinary channel protocols, and those still want
	// the channel-based descriptions.
	mixed := root
	mixed.Protocols = append(append([]fidlgen.Protocol{}, root.Protocols...), fidlgen.Protocol{})
	if IsSyscallLibrary(mixed) {
		t.Error("a library mixing channel and syscall protocols was treated as the kernel ABI")
	}
}

// TestGolden renders the whole description, which pins down the output format
// as well as the translation of every construct the test library uses.
func TestGolden(t *testing.T) {
	generator := NewGenerator()
	got, err := generator.ExecuteTemplate("GenerateSyscallDescriptions",
		compileSyscalls(readTestIR(t), defaultOptions()))
	if err != nil {
		t.Fatalf("failed to render descriptions: %v", err)
	}
	if string(got) != testGolden {
		t.Errorf("generated descriptions differ from %s; got:\n%s", testGoldenPath, got)
	}
}

// syscall returns the compiled syscall of the given name.
func syscall(t *testing.T, root SyscallRoot, name string) Syscall {
	t.Helper()
	for _, group := range root.Groups {
		for _, sc := range group.Syscalls {
			if sc.Name == name {
				return sc
			}
		}
	}
	t.Fatalf("syscall %s was not generated", name)
	return Syscall{}
}

func TestSyscallSignatures(t *testing.T) {
	root := compileSyscalls(readTestIR(t), defaultOptions())
	for _, test := range []struct {
		name string
		want string
	}{
		{
			// Vectors decay into a pointer and a length, out parameters are
			// written through a pointer, and a count named after a buffer ties
			// itself to it.
			name: "zx_channel_read",
			want: "zx_channel_read(handle zx_chan, options int32, bytes ptr[out, array[int8]], " +
				"handles ptr[out, array[zx_handle]], num_bytes len[bytes], num_handles len[handles], " +
				"actual_bytes ptr[out, int32], actual_handles ptr[out, int32])",
		},
		{
			// Aggregates cross the syscall boundary by reference.
			name: "zx_port_queue",
			want: "zx_port_queue(handle zx_port, packet ptr[in, zx_port_packet])",
		},
		{
			// A bits declaration becomes flags, without the base type that
			// syzkaller only wants outside of argument position.
			name: "zx_handle_duplicate",
			want: "zx_handle_duplicate(handle zx_handle, rights flags[zx_rights], out ptr[out, zx_handle])",
		},
		{
			// A @wrapped_return of an aliased time is a resource other
			// syscalls consume; a plain integer return is not.
			name: "zx_deadline_after",
			want: "zx_deadline_after(nanoseconds int64) zx_time",
		},
		{
			name: "zx_ticks_get",
			want: "zx_ticks_get() (ignore_return)",
		},
		{
			// A syscall that never returns would take the executor with it.
			name: "zx_process_exit",
			want: "zx_process_exit(retcode int64) (disabled)",
		},
	} {
		if got := syscall(t, root, test.name).Render(); got != test.want {
			t.Errorf("%s:\n got: %s\nwant: %s", test.name, got, test.want)
		}
	}
}

func TestHandleSubtypeInference(t *testing.T) {
	// zx/Channel.Create declares its outputs as bare handles; without them
	// being channels, syzkaller could not pass one to zx_channel_write.
	inferred := compileSyscalls(readTestIR(t), defaultOptions())
	want := "zx_channel_create(options int32, out0 ptr[out, zx_chan], out1 ptr[out, zx_chan])"
	if got := syscall(t, inferred, "zx_channel_create").Render(); got != want {
		t.Errorf("with inference:\n got: %s\nwant: %s", got, want)
	}

	opts := defaultOptions()
	opts.InferHandleSubtypes = false
	literal := compileSyscalls(readTestIR(t), opts)
	want = "zx_channel_create(options int32, out0 ptr[out, zx_handle], out1 ptr[out, zx_handle])"
	if got := syscall(t, literal, "zx_channel_create").Render(); got != want {
		t.Errorf("without inference:\n got: %s\nwant: %s", got, want)
	}
	// With nothing producing a channel, the resource is unusable to syzkaller
	// and every use of it degrades to an untyped handle.
	for _, res := range literal.Resources {
		if res.Name == "zx_chan" {
			t.Error("zx_chan was declared even though no syscall can produce one")
		}
	}
}

func TestCategoryFiltering(t *testing.T) {
	// zx/Clockfuncs.ClockGetMonotonicViaKernel is @internal.
	const internal = "zx_clock_get_monotonic_via_kernel"
	root := compileSyscalls(readTestIR(t), defaultOptions())
	for _, group := range root.Groups {
		for _, sc := range group.Syscalls {
			if sc.Name == internal {
				t.Fatalf("%s is @internal and should not be generated by default", internal)
			}
		}
	}
	opts := defaultOptions()
	opts.IncludeInternal = true
	syscall(t, compileSyscalls(readTestIR(t), opts), internal)
}

func TestDeclarationsAreUsed(t *testing.T) {
	// syzkaller rejects a declaration that nothing references, so only the
	// reachable ones may be emitted.
	root := compileSyscalls(readTestIR(t), defaultOptions())
	var rendered strings.Builder
	for _, group := range root.Groups {
		for _, sc := range group.Syscalls {
			rendered.WriteString(sc.Render())
			rendered.WriteByte('\n')
		}
	}
	for _, s := range root.Structs {
		rendered.WriteString(s.Render())
	}
	body := rendered.String()
	for _, name := range namesOf(root) {
		if !strings.Contains(body, name) {
			t.Errorf("%s is declared but never used", name)
		}
	}
}

func namesOf(root SyscallRoot) []string {
	var names []string
	for _, f := range root.Flags {
		names = append(names, f.Name)
	}
	for _, s := range root.Structs {
		names = append(names, s.Name)
	}
	for _, r := range root.Resources {
		// The root handle resource is only named as another resource's base.
		if r.Name != zxHandleResource {
			names = append(names, r.Name)
		}
	}
	return names
}
