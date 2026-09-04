// Copyright 2018 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package codegen

import (
	"embed"
	"text/template"

	"github.com/Radon10043/cloud/experiment/fidlgen_syzkaller/src/internal/fidlgen"
)

//go:embed *.tmpl
var templates embed.FS

type Generator struct {
	*fidlgen.Generator
}

func NewGenerator() Generator {
	return Generator{fidlgen.NewGenerator("SyzkallerTemplates", templates,
		fidlgen.NewFormatter(""), template.FuncMap{})}
}

func (g Generator) GenerateSyscallDescription(filename string, root fidlgen.Root, opts Options) error {
	return g.GenerateFile(filename, "GenerateSyscallDescription", compile(root, opts))
}

// GenerateSyscallDescriptions writes the syzkaller descriptions of the Zircon
// system calls declared by root, which must be a library of protocols over the
// "Syscall" transport (see HasSyscalls).
func (g Generator) GenerateSyscallDescriptions(filename string, root fidlgen.Root, opts SyscallOptions) error {
	return g.GenerateFile(filename, "GenerateSyscallDescriptions", compileSyscalls(root, opts))
}
