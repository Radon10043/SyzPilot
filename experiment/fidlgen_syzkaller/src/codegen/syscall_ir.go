// Copyright 2025 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package codegen

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Radon10043/cloud/experiment/fidlgen_syzkaller/src/internal/fidlgen"
)

// SyscallTransport is the FIDL transport of the protocols in //zircon/vdso
// that describe the Zircon system call interface. Protocols over this
// transport are not FIDL protocols spoken over a channel: each method is a
// vDSO entry point, so they are compiled into syzkaller syscall descriptions
// instead of the channel-based descriptions emitted for regular libraries.
const SyscallTransport = "Syscall"

// IsSyscallLibrary reports whether root describes the kernel ABI, i.e. whether
// every protocol it declares is over the "Syscall" transport.
//
// A single Syscall protocol is not enough: a library may declare one as a test
// fixture among ordinary channel protocols, and such a library still wants the
// channel-based descriptions. //zircon/vdso is uniformly over the transport.
func IsSyscallLibrary(root fidlgen.Root) bool {
	syscalls := false
	for i := range root.Protocols {
		if root.Protocols[i].OverTransport() != SyscallTransport {
			return false
		}
		syscalls = true
	}
	return syscalls
}

// SyscallOptions controls which categories of system calls are emitted.
//
// The vDSO declares more entry points than the public ABI exposes: @internal
// calls back the public wrappers, @testonly calls only exist in test builds,
// and @next calls are only available at the NEXT API level.
type SyscallOptions struct {
	IncludeNext     bool
	IncludeInternal bool
	IncludeTestonly bool
	// InferHandleSubtypes types the untyped `Handle` out-parameters that some
	// vDSO declarations still use after the object the syscall creates, so
	// that syzkaller can chain the handle into the syscalls that consume it.
	InferHandleSubtypes bool
}

// SyscallResource is a syzkaller resource declaration, e.g.
// "resource zx_chan[zx_handle]".
type SyscallResource struct {
	Name string
	Base string
	// Values holds the special values syzkaller should occasionally use for
	// this resource, e.g. the invalid handle.
	Values []string
	// base is the type a use of this resource degrades to when the resource
	// turns out not to be both produced and consumed by some syscall.
	base Type
}

// SyscallFlags is a syzkaller flags declaration, e.g. "zx_rights = 1, 2, 4".
type SyscallFlags struct {
	Name string
	// Comment names the FIDL members behind the values, which are emitted
	// numerically so that the description needs no C headers.
	Comment string
	Values  []string
}

// SyscallStruct is a syzkaller struct or union holding a type passed by
// pointer to a system call.
type SyscallStruct struct {
	Name    string
	Comment string
	Members []StructMember
	// IsUnion selects syzkaller's union syntax ("[...]") over struct syntax.
	IsUnion bool

	refStructs []string
	refFlags   []string
	refRes     []string
}

// SyscallArg is a single argument of a system call.
type SyscallArg struct {
	Name string
	Type Type
}

// Syscall is a single Zircon system call.
type Syscall struct {
	// Name is the C name of the syscall, e.g. "zx_channel_read".
	Name string
	Args []SyscallArg
	// Ret is the syzkaller return type, empty when the syscall returns a
	// status (or nothing syzkaller can use).
	Ret Type
	// Attrs are syzkaller call attributes such as "disabled".
	Attrs []string
	// Comment documents why the call is disabled, if it is.
	Comment string

	argInfo []argRefs
}

// argRefs records what a single argument's type refers to, so that resource
// producer/consumer analysis can run once the whole library is compiled.
type argRefs struct {
	dir        direction
	refStructs []string
	refFlags   []string
	refRes     []string
}

type direction int

const (
	dirIn direction = iota
	dirOut
	dirInOut
)

func (d direction) String() string {
	switch d {
	case dirOut:
		return "out"
	case dirInOut:
		return "inout"
	default:
		return "in"
	}
}

// SyscallGroup collects the system calls declared by a single FIDL protocol.
type SyscallGroup struct {
	// Name is the FIDL protocol name, used as a section header.
	Name string
	// Source is the .fidl file the protocol was declared in.
	Source   string
	Syscalls []Syscall
}

// SyscallRoot is the root of the syzkaller IR for a library of system calls.
type SyscallRoot struct {
	// Experiments that have been enabled upstream in fidlc.
	Experiments fidlgen.Experiments

	// Library is the name of the FIDL library, e.g. "zx".
	Library string

	Resources []SyscallResource
	Flags     []SyscallFlags
	Structs   []SyscallStruct
	Groups    []SyscallGroup

	// Warnings records constructs that could not be translated faithfully.
	// They are emitted as comments so the gaps are visible in the output.
	Warnings []string
}

// syscallPrimitiveTypes maps FIDL primitives onto syzkaller's integer types.
// syzkaller has no unsigned types: signedness only affects how it mutates a
// value, and the wire size is what matters here.
var syscallPrimitiveTypes = map[fidlgen.PrimitiveSubtype]Type{
	fidlgen.Bool:                    "int8",
	fidlgen.Int8:                    "int8",
	fidlgen.Uint8:                   "int8",
	fidlgen.ZxExperimentalUchar:     "int8",
	fidlgen.Int16:                   "int16",
	fidlgen.Uint16:                  "int16",
	fidlgen.Int32:                   "int32",
	fidlgen.Uint32:                  "int32",
	fidlgen.Int64:                   "int64",
	fidlgen.Uint64:                  "int64",
	fidlgen.Float32:                 "int32",
	fidlgen.Float64:                 "int64",
	fidlgen.ZxExperimentalUsize64:   "intptr",
	fidlgen.ZxExperimentalUintptr64: "intptr",
}

// syscallAliasTypes maps zx type aliases onto syzkaller resources, so that a
// value produced by one syscall is fed back into the syscalls that consume it.
// Aliases that are absent from this map degrade to their underlying primitive.
var syscallAliasTypes = map[string]Type{
	"zx/Time":        "zx_time",
	"zx/InstantMono": "zx_time",
	"zx/InstantBoot": "zx_time",
	"zx/Vaddr":       "zx_vaddr",
}

// syscallAliasResources declares the resources named by syscallAliasTypes.
var syscallAliasResources = map[Type]SyscallResource{
	"zx_time": {
		Name: "zx_time", Base: "int64", base: "int64",
		// 0 is "already expired" and INT64_MAX is ZX_TIME_INFINITE.
		Values: []string{"0", "0x7fffffffffffffff"},
	},
	"zx_vaddr": {Name: "zx_vaddr", Base: "intptr", base: "intptr"},
}

// zxHandleResource is the root of the handle resource hierarchy; every other
// handle resource derives from it, and it derives from a plain integer.
const zxHandleResource = "zx_handle"

type syscallCompiler struct {
	decls   fidlgen.DeclInfoMap
	structs map[fidlgen.EncodedCompoundIdentifier]fidlgen.Struct
	unions  map[fidlgen.EncodedCompoundIdentifier]fidlgen.Union
	enums   map[fidlgen.EncodedCompoundIdentifier]fidlgen.Enum
	bits    map[fidlgen.EncodedCompoundIdentifier]fidlgen.Bits
	opts    SyscallOptions

	// nameOwner guards against two declarations claiming the same syzkaller
	// identifier: syzkaller has a single namespace for resources, structs and
	// flags, while FIDL does not.
	nameOwner map[string]string

	resources   map[string]*SyscallResource
	flags       map[string]*SyscallFlags
	structs2syz map[fidlgen.EncodedCompoundIdentifier]string
	structsOut  map[string]*SyscallStruct

	// refs accumulates what the type currently being compiled refers to.
	refs *argRefs

	// protocolName and syscallName name the syscall being compiled, and feed
	// the handle subtype inference.
	protocolName string
	syscallName  string

	warnings []string
}

// warn records a translation gap. Warnings are deduplicated and emitted as
// comments at the top of the generated file.
func (c *syscallCompiler) warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	for _, w := range c.warnings {
		if w == msg {
			return
		}
	}
	c.warnings = append(c.warnings, msg)
}

// uniqueName returns a syzkaller identifier for owner, appending suffix if
// base is already claimed by a different declaration.
func (c *syscallCompiler) uniqueName(base, suffix, owner string) string {
	name := base
	for i := 0; ; i++ {
		cur, taken := c.nameOwner[name]
		if !taken {
			c.nameOwner[name] = owner
			return name
		}
		if cur == owner {
			return name
		}
		if i == 0 {
			name = base + suffix
		} else {
			name = fmt.Sprintf("%s%s%d", base, suffix, i)
		}
	}
}

func syzDeclName(eci fidlgen.EncodedCompoundIdentifier) string {
	return "zx_" + fidlgen.ToSnakeCase(string(eci.Parse().Name))
}

// resourceFor registers, on first use, the resource backing a handle subtype.
func (c *syscallCompiler) resourceFor(subtype fidlgen.HandleSubtype) Type {
	name, ok := handleSubtypes[subtype]
	if !ok {
		c.warn("unknown handle subtype %q, treated as an untyped handle", subtype)
		name = zxHandleResource
	}
	if _, ok := c.resources[name]; !ok {
		res := &SyscallResource{Name: name, Base: zxHandleResource, base: zxHandleResource}
		if name == zxHandleResource {
			// ZX_HANDLE_INVALID is 0; teaching syzkaller about it lets it
			// exercise the invalid-handle paths.
			res.Base, res.base, res.Values = "int32", "int32", []string{"0"}
		}
		c.nameOwner[name] = "resource:" + name
		c.resources[name] = res
	}
	c.refs.refRes = append(c.refs.refRes, name)
	return Type(name)
}

// aliasResourceFor registers, on first use, a resource standing for a zx type
// alias such as zx/Time.
func (c *syscallCompiler) aliasResourceFor(name Type) Type {
	if _, ok := c.resources[string(name)]; !ok {
		res := syscallAliasResources[name]
		c.nameOwner[string(name)] = "resource:" + string(name)
		c.resources[string(name)] = &res
	}
	c.refs.refRes = append(c.refs.refRes, string(name))
	return name
}

// flagsFor registers, on first use, the syzkaller flags standing for a FIDL
// enum or bits declaration. Values are emitted numerically: the C spellings of
// these constants are not derivable from FIDL, and numbers keep the generated
// description free of any dependency on Zircon headers.
func (c *syscallCompiler) flagsFor(eci fidlgen.EncodedCompoundIdentifier) (string, bool) {
	owner := "flags:" + string(eci)
	var values, members []string
	if e, ok := c.enums[eci]; ok {
		for _, m := range e.Members {
			values = append(values, m.Value.Value)
			members = append(members, fmt.Sprintf("%s=%s", m.Name, m.Value.Value))
		}
	} else if b, ok := c.bits[eci]; ok {
		for _, m := range b.Members {
			values = append(values, m.Value.Value)
			members = append(members, fmt.Sprintf("%s=%s", m.Name, m.Value.Value))
		}
	} else {
		return "", false
	}
	values = dedupe(values)
	if len(values) == 0 {
		// syzkaller rejects empty flags.
		return "", false
	}
	name := c.uniqueName(syzDeclName(eci), "_flags", owner)
	if _, ok := c.flags[name]; !ok {
		c.flags[name] = &SyscallFlags{
			Name:    name,
			Comment: fmt.Sprintf("%s: %s", eci, strings.Join(members, ", ")),
			Values:  values,
		}
	}
	c.refs.refFlags = append(c.refs.refFlags, name)
	return name, true
}

// structFor registers, on first use, the syzkaller struct or union standing
// for a FIDL struct or union, recursively compiling its members.
func (c *syscallCompiler) structFor(eci fidlgen.EncodedCompoundIdentifier) Type {
	if name, ok := c.structs2syz[eci]; ok {
		c.refs.refStructs = append(c.refs.refStructs, name)
		return Type(name)
	}
	name := c.uniqueName(syzDeclName(eci), "_t", "struct:"+string(eci))
	// Register the name before compiling members so that a self-referential
	// declaration terminates.
	c.structs2syz[eci] = name
	out := &SyscallStruct{Name: name, Comment: string(eci)}
	c.structsOut[name] = out

	// Members are compiled against a scratch reference set that becomes the
	// struct's own dependency list.
	outer := c.refs
	inner := &argRefs{}
	c.refs = inner
	if s, ok := c.structs[eci]; ok {
		for _, m := range s.Members {
			out.Members = append(out.Members, StructMember{
				Name: syzFieldName(m.Name),
				Type: c.syzType(m.Type, m.MaybeFromAlias, m.GetAttributes(), dirIn),
			})
		}
	} else if u, ok := c.unions[eci]; ok {
		out.IsUnion = true
		for _, m := range u.Members {
			out.Members = append(out.Members, StructMember{
				Name: syzFieldName(m.Name),
				Type: c.syzType(m.Type, m.MaybeFromAlias, m.GetAttributes(), dirIn),
			})
		}
	} else {
		c.warn("no declaration found for %s; emitted as an opaque byte", eci)
	}
	if len(out.Members) == 0 {
		// Both syzkaller structs and unions need at least one member. An empty
		// FIDL struct here means the type could not be expressed in FIDL yet
		// (see the TODOs in //zircon/vdso), not that it is empty in C.
		out.IsUnion = false
		out.Comment += " (opaque: the FIDL declaration has no members)"
		out.Members = []StructMember{{Name: "opaque", Type: "int8"}}
	}
	c.refs = outer
	out.refStructs, out.refFlags, out.refRes = inner.refStructs, inner.refFlags, inner.refRes
	c.refs.refStructs = append(c.refs.refStructs, name)
	return Type(name)
}

// syzlangKeywords are the words syzkaller's scanner will not accept as an
// identifier. Zircon parameters such as zx_debuglog_create's `resource` do
// collide with them.
var syzlangKeywords = map[string]bool{
	"include":  true,
	"incdir":   true,
	"define":   true,
	"resource": true,
}

// syzFieldName is the syzkaller name of a FIDL struct member or syscall
// parameter. Names are kept as they are spelled in the vDSO's C interface,
// except where syzlang reserves the word.
func syzFieldName(name fidlgen.Identifier) string {
	str := fidlgen.ToSnakeCase(string(name))
	if syzlangKeywords[str] {
		return str + "_"
	}
	return str
}

// syzType renders the syzkaller type for a FIDL type. dir only matters for
// pointers, which are the only types that carry a direction in syzlang.
func (c *syscallCompiler) syzType(typ fidlgen.Type, from *fidlgen.PartialTypeConstructor, attrs fidlgen.Attributes, dir direction) Type {
	// A zx alias such as zx/Time carries more meaning than the primitive it
	// resolves to, so it wins over the resolved type.
	if from != nil {
		if alias, ok := syscallAliasTypes[string(from.Name)]; ok {
			return c.aliasResourceFor(alias)
		}
	}
	switch typ.Kind {
	case fidlgen.PrimitiveType:
		return c.primitive(typ.PrimitiveSubtype)
	case fidlgen.HandleType:
		return c.resourceFor(typ.HandleSubtype)
	case fidlgen.ArrayType:
		return Type(fmt.Sprintf("array[%s, %d]",
			c.syzType(*typ.ElementType, nil, fidlgen.Attributes{}, dir), *typ.ElementCount))
	case fidlgen.StringArray:
		return Type(fmt.Sprintf("array[int8, %d]", *typ.ElementCount))
	case fidlgen.VectorType:
		// Vectors only reach here nested inside another type; as syscall
		// parameters they decay into a pointer and a length.
		return Type(fmt.Sprintf("array[%s]",
			c.syzType(*typ.ElementType, nil, fidlgen.Attributes{}, dir)))
	case fidlgen.ZxExperimentalPointerType:
		return Type(fmt.Sprintf("ptr[%s, %s]", dir, c.pointee(*typ.PointeeType, attrs, dir)))
	case fidlgen.IdentifierType:
		return c.identifier(typ, dir)
	}
	c.warn("unsupported type kind %q, emitted as an integer", typ.Kind)
	return "intptr"
}

func (c *syscallCompiler) primitive(subtype fidlgen.PrimitiveSubtype) Type {
	if t, ok := syscallPrimitiveTypes[subtype]; ok {
		return t
	}
	c.warn("unknown primitive subtype %q, emitted as intptr", subtype)
	return "intptr"
}

// pointee renders what a pointer parameter points at. A @voidptr pointer is a
// C void*, which syzkaller models as a pointer to a variable-length buffer.
func (c *syscallCompiler) pointee(typ fidlgen.Type, attrs fidlgen.Attributes, dir direction) Type {
	if attrs.HasAttribute("voidptr") {
		return "array[int8]"
	}
	return c.syzType(typ, nil, fidlgen.Attributes{}, dir)
}

func (c *syscallCompiler) identifier(typ fidlgen.Type, dir direction) Type {
	declInfo, ok := c.decls[typ.Identifier]
	if !ok {
		c.warn("unknown identifier %s, emitted as an integer", typ.Identifier)
		return "intptr"
	}
	switch declInfo.Type {
	case fidlgen.EnumDeclType:
		if name, ok := c.flagsFor(typ.Identifier); ok {
			return Type(fmt.Sprintf("flags[%s, %s]", name, c.primitive(c.enums[typ.Identifier].Type)))
		}
		return c.primitive(c.enums[typ.Identifier].Type)
	case fidlgen.BitsDeclType:
		underlying := c.primitive(c.bits[typ.Identifier].Type.PrimitiveSubtype)
		if name, ok := c.flagsFor(typ.Identifier); ok {
			return Type(fmt.Sprintf("flags[%s, %s]", name, underlying))
		}
		return underlying
	case fidlgen.StructDeclType, fidlgen.UnionDeclType:
		return c.structFor(typ.Identifier)
	}
	c.warn("declaration %s of kind %q is not supported in syscall descriptions", typ.Identifier, declInfo.Type)
	return "intptr"
}

// lengthName mirrors the naming zither gives the length parameter a vector
// decays into, so that the generated syscall matches the C vDSO signature.
func lengthName(name string) string {
	if strings.HasSuffix(name, "s") {
		return "num_" + name
	}
	return name + "_size"
}

// params appends the parameters contributed by one payload struct to sc.
//
// Request members are passed by value unless they are annotated @out or
// @inout; response members are always returned through a pointer. Vectors
// decay into a pointer plus a length, matching the C vDSO calling convention.
func (c *syscallCompiler) params(payload fidlgen.Struct, request bool, sc *Syscall) {
	for _, m := range payload.Members {
		attrs := m.GetAttributes()
		dir := dirIn
		switch {
		case !request || attrs.HasAttribute("out"):
			dir = dirOut
		case attrs.HasAttribute("inout"):
			dir = dirInOut
		}
		name := syzFieldName(m.Name)
		refs := &argRefs{dir: dir}
		c.refs = refs

		switch m.Type.Kind {
		case fidlgen.VectorType:
			// A vector decays into a pointer and a length, as it does in the
			// C vDSO signature.
			elem := c.syzType(*m.Type.ElementType, nil, fidlgen.Attributes{}, dir)
			c.addArg(sc, name, Type(fmt.Sprintf("ptr[%s, array[%s]]", dir, elem)), refs)
			// The length always reads the buffer, whichever way the data flows.
			c.addArg(sc, lengthName(name), Type(fmt.Sprintf("len[%s]", name)), &argRefs{dir: dirIn})
		case fidlgen.ZxExperimentalPointerType:
			// The pointer itself carries the indirection; @out only sets its
			// direction.
			c.addArg(sc, name, c.syzType(m.Type, m.MaybeFromAlias, attrs, dir), refs)
		case fidlgen.HandleType:
			c.addArg(sc, name, c.handleParam(m, name, dir), refs)
		default:
			t := c.syzType(m.Type, m.MaybeFromAlias, attrs, dir)
			// Aggregates are passed by reference across the syscall boundary,
			// and anything flowing outwards needs somewhere to be written.
			if dir != dirIn || c.isAggregate(m.Type) {
				t = Type(fmt.Sprintf("ptr[%s, %s]", dir, t))
			}
			c.addArg(sc, name, t, refs)
		}
	}
}

// isAggregate reports whether a FIDL type is a struct or a union. The vDSO
// passes those across the syscall boundary by pointer, never by value.
func (c *syscallCompiler) isAggregate(typ fidlgen.Type) bool {
	if typ.Kind != fidlgen.IdentifierType {
		return false
	}
	info, ok := c.decls[typ.Identifier]
	if !ok {
		return false
	}
	return info.Type == fidlgen.StructDeclType || info.Type == fidlgen.UnionDeclType
}

// handleParam compiles a handle parameter, inferring the object type of an
// untyped out-parameter when asked to.
func (c *syscallCompiler) handleParam(m fidlgen.StructMember, name string, dir direction) Type {
	subtype := m.Type.HandleSubtype
	if c.opts.InferHandleSubtypes && subtype == fidlgen.HandleSubtypeNone {
		if inferred, ok := c.inferHandleSubtype(name, dir); ok {
			c.warn("some untyped handle parameters were typed after the object" +
				" their syscall operates on; pass -infer-handle-subtypes=false" +
				" to keep them untyped")
			subtype = inferred
		}
	}
	t := c.resourceFor(subtype)
	if dir != dirIn {
		t = Type(fmt.Sprintf("ptr[%s, %s]", dir, t))
	}
	return t
}

// handleSubtypeByName indexes the handle subtypes syzkaller has a resource for
// by their FIDL spelling.
var handleSubtypeByName = func() map[string]fidlgen.HandleSubtype {
	m := make(map[string]fidlgen.HandleSubtype, len(handleSubtypes))
	for subtype := range handleSubtypes {
		m[string(subtype)] = subtype
	}
	return m
}()

// inferHandleSubtype guesses which kind of object an untyped handle parameter
// refers to. Several vDSO declarations still spell a parameter as a bare
// `Handle` even though the syscall only ever accepts or produces one kind of
// object; taken literally, they would leave syzkaller unable to feed a handle
// from the syscall that creates it into the syscalls that use it.
//
// Zircon names a syscall after the object it acts on, which is what the guess
// rests on: zx_<obj>_<op>(handle, ...) takes an <obj>, and zx_<obj>_create
// produces one. An input is only eligible when it is spelled exactly "handle",
// so that a second handle such as zx_vmar_unmap_handle_close_thread_exit's
// close_handle stays untyped. The parameter name wins over the syscall, so
// that zx_msi_create's out_interrupt is an interrupt and not an MSI object.
func (c *syscallCompiler) inferHandleSubtype(param string, dir direction) (fidlgen.HandleSubtype, bool) {
	if dir == dirIn {
		if param != "handle" {
			return fidlgen.HandleSubtypeNone, false
		}
	} else {
		hint := strings.TrimPrefix(param, "out_")
		hint = strings.TrimSuffix(hint, "_out")
		hint = strings.TrimRight(hint, "0123456789")
		hint = strings.TrimSuffix(hint, "_")
		if subtype, ok := handleSubtypeByName[hint]; ok {
			// A parameter that names itself a plain handle really is untyped.
			return subtype, subtype != fidlgen.HandleSubtypeNone
		}
	}
	if subtype, ok := handleSubtypeByName[c.protocolName]; ok {
		return subtype, true
	}
	// Protocols carrying @no_protocol_prefix are named after their .fidl file
	// rather than the object, so fall back to the syscall name itself.
	family := strings.TrimPrefix(c.syscallName, "zx_")
	if dir == dirIn {
		family, _, _ = strings.Cut(family, "_")
	} else {
		trimmed := strings.TrimSuffix(family, "_create")
		if trimmed == family {
			return fidlgen.HandleSubtypeNone, false
		}
		family = trimmed
	}
	if subtype, ok := handleSubtypeByName[family]; ok {
		return subtype, subtype != fidlgen.HandleSubtypeNone
	}
	return fidlgen.HandleSubtypeNone, false
}

var flagsWithBase = regexp.MustCompile(`^flags\[([^,\[\]]+), \w+\]$`)

func (c *syscallCompiler) addArg(sc *Syscall, name string, typ Type, refs *argRefs) {
	// Syscall arguments are register-sized, so syzkaller takes flags there
	// without the base type it requires everywhere else.
	typ = Type(flagsWithBase.ReplaceAllString(string(typ), "flags[$1]"))
	sc.Args = append(sc.Args, SyscallArg{Name: name, Type: typ})
	sc.argInfo = append(sc.argInfo, *refs)
}

var lengthSuffixes = []string{"_size", "_len", "_count", "_bytes"}

var pointerType = regexp.MustCompile(`^ptr\[(in|out|inout), (.*)\]$`)

// linkLengths ties an integer parameter that holds the size of a buffer to
// that buffer with a syzkaller len[] expression, so that syzkaller keeps the
// two consistent instead of generating a length unrelated to the allocation.
//
// Only vectors say so in FIDL; the rest of the vDSO passes a bare pointer
// alongside a separately declared count, named num_<buffer>, <buffer>_size and
// so on. A pointer that turns out to have a length is a buffer of many
// elements rather than a pointer to one, so it becomes an array.
func linkLengths(sc *Syscall) {
	buffers := make(map[string]int)
	for i, arg := range sc.Args {
		if pointerType.MatchString(string(arg.Type)) {
			buffers[arg.Name] = i
		}
	}
	for i, arg := range sc.Args {
		if !isPlainInteger(arg.Type) {
			continue
		}
		target, ok := "", false
		if rest := strings.TrimPrefix(arg.Name, "num_"); rest != arg.Name {
			target, ok = rest, true
		} else {
			for _, suffix := range lengthSuffixes {
				if base := strings.TrimSuffix(arg.Name, suffix); base != arg.Name {
					target, ok = base, true
					break
				}
			}
		}
		buf, isBuffer := buffers[target]
		if !ok || !isBuffer {
			continue
		}
		sc.Args[i].Type = Type(fmt.Sprintf("len[%s]", target))
		sc.Args[buf].Type = asBuffer(sc.Args[buf].Type)
	}
}

// asBuffer turns a pointer to a single value into a pointer to an array of
// them, leaving pointers that already point at an array alone.
func asBuffer(t Type) Type {
	m := pointerType.FindStringSubmatch(string(t))
	if m == nil || strings.HasPrefix(m[2], "array[") {
		return t
	}
	return Type(fmt.Sprintf("ptr[%s, array[%s]]", m[1], m[2]))
}

func isPlainInteger(t Type) bool {
	switch t {
	case "int8", "int16", "int32", "int64", "intptr":
		return true
	}
	return false
}

// syscallName derives the C name of a syscall the way zither does: the
// protocol name prefixes the method name unless the protocol opts out with
// @no_protocol_prefix.
func syscallName(protocol fidlgen.Protocol, method fidlgen.Method) string {
	name := string(method.Name)
	if !protocol.HasAttribute("no_protocol_prefix") {
		name = string(protocol.Name.Parse().Name) + name
	}
	return "zx_" + fidlgen.ToSnakeCase(name)
}

// skip reports whether a syscall is outside the requested categories.
func (c *syscallCompiler) skip(method fidlgen.Method) bool {
	attrs := method.GetAttributes()
	if attrs.HasAttribute("testonly") || attrs.HasAttribute("test_category1") ||
		attrs.HasAttribute("test_category2") {
		return !c.opts.IncludeTestonly
	}
	if attrs.HasAttribute("internal") {
		return !c.opts.IncludeInternal
	}
	if attrs.HasAttribute("next") {
		return !c.opts.IncludeNext
	}
	return false
}

// payloadStruct resolves a method payload type to its struct declaration.
func (c *syscallCompiler) payloadStruct(typ *fidlgen.Type) (fidlgen.Struct, bool) {
	if typ == nil || typ.Kind != fidlgen.IdentifierType {
		return fidlgen.Struct{}, false
	}
	s, ok := c.structs[typ.Identifier]
	return s, ok
}

func (c *syscallCompiler) compileMethod(protocol fidlgen.Protocol, method fidlgen.Method) (Syscall, bool) {
	if c.skip(method) {
		return Syscall{}, false
	}
	sc := Syscall{Name: syscallName(protocol, method)}
	c.protocolName = fidlgen.ToSnakeCase(string(protocol.Name.Parse().Name))
	c.syscallName = sc.Name

	if request, ok := c.payloadStruct(method.RequestPayload); ok {
		c.params(request, true, &sc)
	}

	// With error syntax the syscall returns zx_status_t and the success
	// variant carries the out parameters; without it the response describes
	// the C return value instead.
	response := method.ResponsePayload
	if method.ValueType != nil {
		response = method.ValueType
	}
	if s, ok := c.payloadStruct(response); ok && method.HasResponse {
		switch {
		case s.HasAttribute("wrapped_return"):
			if len(s.Members) == 1 {
				sc.Ret = c.returnType(s.Members[0])
			}
			if sc.Ret == "" {
				// The syscall returns a plain value that no other syscall
				// consumes, so there is nothing for syzkaller to track.
				sc.Attrs = append(sc.Attrs, "ignore_return")
			}
		case !method.HasError && !s.IsAnonymous():
			// A named response struct is returned by value, which syzkaller
			// cannot express.
			sc.Attrs = append(sc.Attrs, "ignore_return")
		default:
			c.params(s, false, &sc)
		}
	}

	linkLengths(&sc)

	if method.HasAttribute("noreturn") {
		// A syscall that never returns terminates the executor, so syzkaller
		// must know about it but must not call it.
		sc.Attrs = append(sc.Attrs, "disabled")
		sc.Comment = "does not return"
	}
	return sc, true
}

// returnType maps a @wrapped_return member onto a syzkaller return type.
// Only values syzkaller can feed back into other syscalls are worth naming;
// the rest are marked ignore_return.
func (c *syscallCompiler) returnType(m fidlgen.StructMember) Type {
	refs := &argRefs{dir: dirOut}
	c.refs = refs
	t := c.syzType(m.Type, m.MaybeFromAlias, m.GetAttributes(), dirOut)
	if _, ok := c.resources[string(t)]; !ok {
		return ""
	}
	// Record the return value as a producer of the resource.
	return t
}

func dedupe(values []string) []string {
	seen := make(map[string]bool, len(values))
	var out []string
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// compileSyscalls turns a library of "Syscall"-transport protocols into
// syzkaller syscall descriptions.
func compileSyscalls(fidlData fidlgen.Root, opts SyscallOptions) SyscallRoot {
	fidlData = fidlData.ForBindings("syzkaller")
	c := &syscallCompiler{
		decls:       fidlData.DeclInfo(),
		structs:     make(map[fidlgen.EncodedCompoundIdentifier]fidlgen.Struct),
		unions:      make(map[fidlgen.EncodedCompoundIdentifier]fidlgen.Union),
		enums:       make(map[fidlgen.EncodedCompoundIdentifier]fidlgen.Enum),
		bits:        make(map[fidlgen.EncodedCompoundIdentifier]fidlgen.Bits),
		opts:        opts,
		nameOwner:   make(map[string]string),
		resources:   make(map[string]*SyscallResource),
		flags:       make(map[string]*SyscallFlags),
		structs2syz: make(map[fidlgen.EncodedCompoundIdentifier]string),
		structsOut:  make(map[string]*SyscallStruct),
		refs:        &argRefs{},
	}
	for _, v := range fidlData.Structs {
		c.structs[v.Name] = v
	}
	for _, v := range fidlData.ExternalStructs {
		c.structs[v.Name] = v
	}
	for _, v := range fidlData.Unions {
		c.unions[v.Name] = v
	}
	for _, v := range fidlData.Enums {
		c.enums[v.Name] = v
	}
	for _, v := range fidlData.Bits {
		c.bits[v.Name] = v
	}

	root := SyscallRoot{
		Experiments: fidlData.Experiments,
		Library:     string(fidlData.Name),
	}
	for _, protocol := range fidlData.Protocols {
		if protocol.OverTransport() != SyscallTransport {
			continue
		}
		group := SyscallGroup{
			Name:   string(protocol.Name),
			Source: strings.TrimPrefix(protocol.Location.Filename, "../../"),
		}
		for _, method := range protocol.Methods {
			if sc, ok := c.compileMethod(protocol, method); ok {
				group.Syscalls = append(group.Syscalls, sc)
			}
		}
		if len(group.Syscalls) > 0 {
			root.Groups = append(root.Groups, group)
		}
	}

	c.demoteUnusableResources(&root)
	c.collect(&root)
	root.Warnings = c.warnings
	return root
}

// resourceUsage summarizes whether a resource is produced and consumed.
type resourceUsage struct{ ctor, input bool }

// demoteUnusableResources replaces resources that no syscall can produce, or
// that no syscall consumes, with their base type. syzkaller rejects such
// resources outright, and a handle Zircon never hands out is of no use to a
// fuzzer anyway.
func (c *syscallCompiler) demoteUnusableResources(root *SyscallRoot) {
	for {
		usage := c.resourceUsage(root)
		demote := ""
		// Sort for a deterministic choice when several resources are unusable.
		names := make([]string, 0, len(c.resources))
		for name := range c.resources {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			u, used := usage[name]
			if !used {
				continue
			}
			if u.ctor && u.input {
				continue
			}
			if c.resources[name].base == Type(name) {
				continue
			}
			demote = name
			break
		}
		if demote == "" {
			return
		}
		base := c.resources[demote].base
		switch {
		case !usage[demote].ctor:
			c.warn("resource %s is never produced by a syscall; using %s instead", demote, base)
		default:
			c.warn("resource %s is never consumed by a syscall; using %s instead", demote, base)
		}
		c.replaceType(root, demote, base)
		delete(c.resources, demote)
	}
}

// resourceUsage walks every syscall argument, through the structs it reaches,
// recording which resources are produced and which are consumed.
func (c *syscallCompiler) resourceUsage(root *SyscallRoot) map[string]resourceUsage {
	usage := make(map[string]resourceUsage)
	mark := func(name string, dir direction) {
		u := usage[name]
		if dir == dirIn || dir == dirInOut {
			u.input = true
		}
		if dir == dirOut || dir == dirInOut {
			u.ctor = true
		}
		usage[name] = u
	}
	for _, group := range root.Groups {
		for _, sc := range group.Syscalls {
			for _, refs := range sc.argInfo {
				for _, name := range c.reachableResources(refs) {
					mark(name, refs.dir)
				}
			}
			if sc.Ret != "" {
				mark(string(sc.Ret), dirOut)
			}
		}
	}
	return usage
}

// reachableResources returns the resources refs names, including those only
// reachable through the structs it names.
func (c *syscallCompiler) reachableResources(refs argRefs) []string {
	var out []string
	out = append(out, refs.refRes...)
	seen := make(map[string]bool)
	queue := append([]string(nil), refs.refStructs...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		s, ok := c.structsOut[name]
		if !ok {
			continue
		}
		out = append(out, s.refRes...)
		queue = append(queue, s.refStructs...)
	}
	return out
}

// replaceType rewrites every occurrence of the syzkaller type named from into
// to, both in syscall signatures and in struct members.
func (c *syscallCompiler) replaceType(root *SyscallRoot, from string, to Type) {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(from) + `\b`)
	sub := func(t Type) Type { return Type(re.ReplaceAllString(string(t), string(to))) }
	rename := func(names []string) []string {
		var out []string
		for _, n := range names {
			if n == from {
				if _, ok := c.resources[string(to)]; ok {
					out = append(out, string(to))
				}
				continue
			}
			out = append(out, n)
		}
		return out
	}
	for gi := range root.Groups {
		for si := range root.Groups[gi].Syscalls {
			sc := &root.Groups[gi].Syscalls[si]
			for ai := range sc.Args {
				sc.Args[ai].Type = sub(sc.Args[ai].Type)
				sc.argInfo[ai].refRes = rename(sc.argInfo[ai].refRes)
			}
			if sc.Ret == Type(from) {
				if _, ok := c.resources[string(to)]; ok {
					sc.Ret = to
				} else {
					sc.Ret = ""
					sc.Attrs = append(sc.Attrs, "ignore_return")
				}
			}
		}
	}
	for _, s := range c.structsOut {
		for mi := range s.Members {
			s.Members[mi].Type = sub(s.Members[mi].Type)
		}
		s.refRes = rename(s.refRes)
	}
}

// collect gathers the declarations the syscalls actually reference. syzkaller
// rejects declarations that nothing uses, so anything unreachable is dropped.
func (c *syscallCompiler) collect(root *SyscallRoot) {
	structs := make(map[string]bool)
	flags := make(map[string]bool)
	resources := make(map[string]bool)

	var visitStruct func(name string)
	visitStruct = func(name string) {
		if structs[name] {
			return
		}
		structs[name] = true
		s, ok := c.structsOut[name]
		if !ok {
			return
		}
		for _, n := range s.refFlags {
			flags[n] = true
		}
		for _, n := range s.refRes {
			resources[n] = true
		}
		for _, n := range s.refStructs {
			visitStruct(n)
		}
	}
	for _, group := range root.Groups {
		for _, sc := range group.Syscalls {
			for _, refs := range sc.argInfo {
				for _, n := range refs.refFlags {
					flags[n] = true
				}
				for _, n := range refs.refRes {
					resources[n] = true
				}
				for _, n := range refs.refStructs {
					visitStruct(n)
				}
			}
			if sc.Ret != "" {
				resources[string(sc.Ret)] = true
			}
		}
	}
	// A resource's base may itself be a resource that nothing names directly.
	for {
		grew := false
		for name := range resources {
			res, ok := c.resources[name]
			if !ok {
				continue
			}
			if _, ok := c.resources[res.Base]; ok && !resources[res.Base] {
				resources[res.Base] = true
				grew = true
			}
		}
		if !grew {
			break
		}
	}

	for name := range resources {
		if res, ok := c.resources[name]; ok {
			root.Resources = append(root.Resources, *res)
		}
	}
	sort.Slice(root.Resources, func(i, j int) bool {
		// Base resources must be declared before those derived from them.
		if root.Resources[i].Name == zxHandleResource {
			return true
		}
		if root.Resources[j].Name == zxHandleResource {
			return false
		}
		return root.Resources[i].Name < root.Resources[j].Name
	})
	for name := range flags {
		if f, ok := c.flags[name]; ok {
			root.Flags = append(root.Flags, *f)
		}
	}
	sort.Slice(root.Flags, func(i, j int) bool { return root.Flags[i].Name < root.Flags[j].Name })
	for name := range structs {
		if s, ok := c.structsOut[name]; ok {
			root.Structs = append(root.Structs, *s)
		}
	}
	sort.Slice(root.Structs, func(i, j int) bool { return root.Structs[i].Name < root.Structs[j].Name })
}

// Render returns the syzkaller declaration of the resource.
func (r SyscallResource) Render() string {
	decl := fmt.Sprintf("resource %s[%s]", r.Name, r.Base)
	if len(r.Values) > 0 {
		decl += ": " + strings.Join(r.Values, ", ")
	}
	return decl
}

// Render returns the syzkaller declaration of the flags.
func (f SyscallFlags) Render() string {
	return fmt.Sprintf("%s = %s", f.Name, strings.Join(f.Values, ", "))
}

// Render returns the syzkaller declaration of the struct or union, with its
// member types aligned into a column.
func (s SyscallStruct) Render() string {
	open, close := "{", "}"
	if s.IsUnion {
		open, close = "[", "]"
	}
	width := 0
	for _, m := range s.Members {
		if len(m.Name) > width {
			width = len(m.Name)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", s.Name, open)
	for _, m := range s.Members {
		fmt.Fprintf(&b, "\t%-*s  %s\n", width, m.Name, m.Type)
	}
	b.WriteString(close)
	return b.String()
}

// Render returns the syzkaller declaration of the system call.
func (s Syscall) Render() string {
	args := make([]string, 0, len(s.Args))
	for _, arg := range s.Args {
		args = append(args, fmt.Sprintf("%s %s", arg.Name, arg.Type))
	}
	decl := fmt.Sprintf("%s(%s)", s.Name, strings.Join(args, ", "))
	if s.Ret != "" {
		decl += " " + string(s.Ret)
	}
	if len(s.Attrs) > 0 {
		decl += " (" + strings.Join(dedupe(s.Attrs), ", ") + ")"
	}
	return decl
}
