// Copyright 2020 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/Radon10043/cloud/experiment/fidlgen_syzkaller/src/codegen"
	"github.com/Radon10043/cloud/experiment/fidlgen_syzkaller/src/internal/fidlgen"
)

type flagsDef struct {
	jsonPath        *string
	outputPath      *string
	outputConstPath *string
	arches          *string
	includeNext     *bool
	includeInternal *bool
	includeTestonly *bool
	inferHandles    *bool
	assumeVdsoRes   *bool
}

var flags = flagsDef{
	jsonPath: flag.String("json", "",
		"relative path to the FIDL intermediate representation."),
	outputPath: flag.String("output-syz", "",
		"output path for the generated syz.txt file."),
	outputConstPath: flag.String("output-syz-const", "",
		"optional output path for the companion syzkaller .const file. The "+
			"generated descriptions inline all constant values, so the file "+
			"only records the architectures they apply to."),
	arches: flag.String("arches", "amd64, arm64",
		"comma-separated syzkaller architectures the descriptions apply to, "+
			"recorded in the .const file."),
	includeNext: flag.Bool("include-next", true,
		"include @next system calls, which are only available at the NEXT API level."),
	includeInternal: flag.Bool("include-internal", false,
		"include @internal system calls, which are not part of the public ABI."),
	includeTestonly: flag.Bool("include-testonly", false,
		"include @testonly system calls, which only exist in test builds."),
	assumeVdsoRes: flag.Bool("assume-vdso-resources", false,
		"in FIDL protocol descriptions, name a resource for every handle "+
			"subtype. This requires the syscall descriptions this tool "+
			"generates from the vDSO to be in use; without them, subtypes "+
			"syzkaller does not declare degrade to an untyped handle."),
	inferHandles: flag.Bool("infer-handle-subtypes", true,
		"type the untyped handle out-parameters that some vDSO declarations "+
			"still use after the object their system call creates."),
}

// valid returns true if the parsed flags are valid.
func (f flagsDef) valid() bool {
	return *f.jsonPath != "" && *f.outputPath != ""
}

func printUsage() {
	program := path.Base(os.Args[0])
	message := `Usage: ` + program + ` [flags]

Syzkaller FIDL backend, used to generate Syzkaller bindings from JSON IR input
(the intermediate representation of a FIDL library).

Libraries whose protocols are declared over the "Syscall" transport (that is,
the Zircon vDSO in //zircon/vdso) describe the kernel's system call interface
rather than a FIDL protocol; for those, the syscall descriptions themselves are
generated. Every other library yields the channel-based descriptions used to
fuzz FIDL protocols.

Flags:
`
	fmt.Fprint(flag.CommandLine.Output(), message)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if !flags.valid() {
		printUsage()
		os.Exit(1)
	}

	root, err := fidlgen.ReadJSONIr(*flags.jsonPath)
	if err != nil {
		log.Fatalf("Failed to read JSON: %v", err)
	}

	generator := codegen.NewGenerator()
	if codegen.IsSyscallLibrary(root) {
		opts := codegen.SyscallOptions{
			IncludeNext:         *flags.includeNext,
			IncludeInternal:     *flags.includeInternal,
			IncludeTestonly:     *flags.includeTestonly,
			InferHandleSubtypes: *flags.inferHandles,
		}
		if err := generator.GenerateSyscallDescriptions(*flags.outputPath, root, opts); err != nil {
			log.Fatalf("Failed to compile syzkaller syscall descriptions: %v", err)
		}
		if *flags.outputConstPath != "" {
			if err := codegen.WriteConstFile(*flags.outputConstPath, *flags.arches); err != nil {
				log.Fatalf("Failed to write syzkaller const file: %v", err)
			}
		}
		return
	}

	opts := codegen.Options{AssumeVdsoResources: *flags.assumeVdsoRes}
	if err := generator.GenerateSyscallDescription(*flags.outputPath, root, opts); err != nil {
		log.Fatalf("Failed to compile syzkaller description: %v", err)
	}
}
