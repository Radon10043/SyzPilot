// Copyright 2018 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package codegen

import (
	"fmt"
	"strings"

	"github.com/Radon10043/cloud/experiment/fidlgen_syzkaller/src/internal/fidlgen"
)

const (
	OutOfLineSuffix = "OutOfLine"
	InLineSuffix    = "InLine"
	RequestSuffix   = "Request"
	ResponseSuffix  = "Response"
	EventSuffix     = "Event"
	HandlesSuffix   = "Handles"
	EnvelopeSuffix  = "Envelope"
)

const (
	// fidlEnvelopeSize is the size of the envelope a table reserves for each
	// of its ordinals.
	fidlEnvelopeSize = 8

	// fidlEnvelopeInlineSize is how much of an envelope holds the value
	// itself. A value that does not fit is stored out-of-line instead, and
	// the envelope describes it.
	fidlEnvelopeInlineSize = 4
)

// Type represents a syzkaller type including type-options.
type Type string

// Enum represents a set of syzkaller flags
type Enum struct {
	Name    string
	Type    string
	Members []string
}

// Bits represents a set of syzkaller flags
type Bits struct {
	Name    string
	Type    string
	Members []string
}

// Struct represents a syzkaller struct.
type Struct struct {
	Name    string
	Members []StructMember
}

// StructMember represents a member of a syzkaller struct.
type StructMember struct {
	Name string
	Type Type
}

// Union represents a syzkaller union.
type Union struct {
	Name    string
	Members []StructMember
	VarLen  bool
}

// Protocol represents a FIDL protocol in terms of syzkaller structures.
type Protocol struct {
	Name string

	// ProtocolNameString is the string service name for this FIDL protocol.
	ProtocolNameString string

	// Methods is a list of methods for this FIDL protocol.
	Methods []Method
}

// Method represents a method of a FIDL protocol in terms of syzkaller syscalls.
type Method struct {
	// Ordinal is the ordinal for this method.
	Ordinal uint64

	// Name is the name of the Method, including the protocol name as a prefix.
	Name string

	// Request represents a struct containing the request parameters.
	Request *Struct

	// RequestHandles represents a struct containing the handles in the request parameters.
	RequestHandles *Struct

	// Response represents an optional struct containing the response parameters.
	Response *Struct

	// ResponseHandles represents a struct containing the handles in the response parameters.
	ResponseHandles *Struct

	// Structs contain the structs this method declares: the request and
	// response messages and their handles, plus everything generated during
	// depth-first traversal of them. They are declared centrally, alongside
	// the library's other structs, so that a name reached by more than one
	// route is only declared once.
	Structs []Struct

	// Unions contain all the unions generated during depth-first traversal of Request/Response.
	Unions []Union
}

// Root is the root of the syzkaller backend IR structure.
type Root struct {
	// Experiments that have been enabled upstream in fidlc.
	Experiments fidlgen.Experiments

	// Name is the name of the library.
	Name string

	// C header file path to be included in syscall description.
	HeaderPath string

	// Protocols represent the list of FIDL protocols represented as a collection of syskaller syscall descriptions.
	Protocols []Protocol

	// Structs correspond to syzkaller structs.
	Structs []Struct

	// ExternalStructs correspond to external syzkaller structs.
	ExternalStructs []Struct

	// Unions correspond to syzkaller unions.
	Unions []Union

	// Enums correspond to syzkaller flags.
	Enums []Enum

	// Bits correspond to syzkaller flags.
	Bits []Bits
}

type StructMap map[fidlgen.EncodedCompoundIdentifier]fidlgen.Struct
type TableMap map[fidlgen.EncodedCompoundIdentifier]fidlgen.Table
type UnionMap map[fidlgen.EncodedCompoundIdentifier]fidlgen.Union
type EnumMap map[fidlgen.EncodedCompoundIdentifier]fidlgen.Enum
type BitsMap map[fidlgen.EncodedCompoundIdentifier]fidlgen.Bits

type compiler struct {
	// decls contains all top-level declarations for the FIDL source.
	decls fidlgen.DeclInfoMap

	// compiling tracks the unions currently on the compile stack, so a
	// self-referential union does not recurse forever.
	compiling map[fidlgen.EncodedCompoundIdentifier]struct{}

	// structs contain all top-level struct definitions for the FIDL source.
	structs StructMap

	// tables contain all top-level table definitions for the FIDL source.
	tables TableMap

	// unions contain all top-level union definitions for the FIDL source.
	unions UnionMap

	// enums contain all top-level enum definitions for the FIDL source.
	enums EnumMap

	// bits contain all top-level bits definitions for the FIDL source.
	bits BitsMap

	// library is the identifier for the current library.
	library fidlgen.LibraryIdentifier

	// opts are the options the descriptions are generated under.
	opts Options
}

// Options controls how FIDL protocol descriptions are generated.
type Options struct {
	// AssumeVdsoResources says that the syscall descriptions generated from
	// the Zircon vDSO are in use alongside these, so every handle subtype has
	// a resource declared for it. Without it, subtypes that syzkaller's own
	// descriptions do not declare degrade to an untyped handle.
	AssumeVdsoResources bool
}

var reservedWords = map[string]struct{}{
	// Keywords the syzkaller scanner recognises (pkg/ast/scanner.go).
	"include":  {},
	"incdir":   {},
	"define":   {},
	"resource": {},

	// Field names the syzkaller compiler reserves (prog/size.go).
	"parent":  {},
	"syscall": {},

	"array":     {},
	"buffer":    {},
	"int8":      {},
	"int16":     {},
	"int32":     {},
	"int64":     {},
	"intptr":    {},
	"ptr":       {},
	"type":      {},
	"len":       {},
	"string":    {},
	"stringnoz": {},
	"const":     {},
	"in":        {},
	"out":       {},
	"flags":     {},
	"bytesize":  {},
	"bitsize":   {},
	"text":      {},
	"void":      {},
}

var primitiveTypes = map[fidlgen.PrimitiveSubtype]string{
	fidlgen.Bool:    "int8",
	fidlgen.Int8:    "int8",
	fidlgen.Int16:   "int16",
	fidlgen.Int32:   "int32",
	fidlgen.Int64:   "int64",
	fidlgen.Uint8:   "int8",
	fidlgen.Uint16:  "int16",
	fidlgen.Uint32:  "int32",
	fidlgen.Uint64:  "int64",
	fidlgen.Float32: "int32",
	fidlgen.Float64: "int64",
}

// zxPrimitiveTypes are the primitives that only the Zircon vDSO uses.
var zxPrimitiveTypes = map[fidlgen.PrimitiveSubtype]string{
	fidlgen.ZxExperimentalUchar:     "int8",
	fidlgen.ZxExperimentalUsize64:   "intptr",
	fidlgen.ZxExperimentalUintptr64: "intptr",
}

var handleSubtypes = map[fidlgen.HandleSubtype]string{
	fidlgen.HandleSubtypeBti:          "zx_bti",
	fidlgen.HandleSubtypeChannel:      "zx_chan",
	fidlgen.HandleSubtypeClock:        "zx_clock",
	fidlgen.HandleSubtypeCounter:      "zx_counter",
	fidlgen.HandleSubtypeDebugLog:     "zx_debuglog",
	fidlgen.HandleSubtypeEvent:        "zx_event",
	fidlgen.HandleSubtypeEventpair:    "zx_eventpair",
	fidlgen.HandleSubtypeException:    "zx_exception",
	fidlgen.HandleSubtypeFifo:         "zx_fifo",
	fidlgen.HandleSubtypeGuest:        "zx_guest",
	fidlgen.HandleSubtypeNone:         "zx_handle",
	fidlgen.HandleSubtypeInterrupt:    "zx_interrupt",
	fidlgen.HandleSubtypeIob:          "zx_iob",
	fidlgen.HandleSubtypeIommu:        "zx_iommu",
	fidlgen.HandleSubtypeJob:          "zx_job",
	fidlgen.HandleSubtypeMsi:          "zx_msi",
	fidlgen.HandleSubtypePager:        "zx_pager",
	fidlgen.HandleSubtypePciDevice:    "zx_pcidevice",
	fidlgen.HandleSubtypePmt:          "zx_pmt",
	fidlgen.HandleSubtypePort:         "zx_port",
	fidlgen.HandleSubtypeProcess:      "zx_process",
	fidlgen.HandleSubtypeProfile:      "zx_profile",
	fidlgen.HandleSubtypeResource:     "zx_resource",
	fidlgen.HandleSubtypeSocket:       "zx_socket",
	fidlgen.HandleSubtypeStream:       "zx_stream",
	fidlgen.HandleSubtypeSuspendToken: "zx_suspendtoken",
	fidlgen.HandleSubtypeThread:       "zx_thread",
	fidlgen.HandleSubtypeTimer:        "zx_timer",
	fidlgen.HandleSubtypeVcpu:         "zx_vcpu",
	fidlgen.HandleSubtypeVmar:         "zx_vmar",
	fidlgen.HandleSubtypeVmo:          "zx_vmo",
}

func isReservedWord(str string) bool {
	_, ok := reservedWords[str]
	return ok
}

func changeIfReserved(val fidlgen.Identifier, ext string) string {
	str := string(val) + ext
	if isReservedWord(str) {
		return str + "_"
	}
	return str
}

func formatLibrary(library fidlgen.LibraryIdentifier, sep string) string {
	parts := []string{}
	for _, part := range library {
		parts = append(parts, string(part))
	}
	return changeIfReserved(fidlgen.Identifier(strings.Join(parts, sep)), "")
}

func formatLibraryPath(library fidlgen.LibraryIdentifier) string {
	return formatLibrary(library, "/")
}

func (c *compiler) compileIdentifier(id fidlgen.Identifier, ext string) string {
	str := string(id)
	str = fidlgen.ToSnakeCase(str)
	return changeIfReserved(fidlgen.Identifier(str), ext)
}

func (c *compiler) compileCompoundIdentifier(eci fidlgen.EncodedCompoundIdentifier, ext string) string {
	val := eci.Parse()
	strs := []string{}
	strs = append(strs, formatLibrary(val.Library, "_"))
	strs = append(strs, changeIfReserved(val.Name, ext))
	return strings.Join(strs, "_")
}

// foreignIntType returns an integer type of the same width as the declaration
// named by id. A dependency library contributes only a DeclInfo to the JSON
// IR - no members, no primitive subtype - so the width recorded in its type
// shape is all there is to go on. Layout is all syzkaller needs here: the
// flags set itself is emitted by the file generated for that library.
func (c *compiler) foreignIntType(id fidlgen.EncodedCompoundIdentifier) Type {
	size := 0
	if info, ok := c.decls[id]; ok && info.TypeShapeV2 != nil {
		size = info.TypeShapeV2.InlineSize
	}
	switch size {
	case 1:
		return Type("int8")
	case 2:
		return Type("int16")
	case 8:
		return Type("int64")
	default:
		return Type("int32")
	}
}

// foreignShape reports whether the declaration named by id carries any
// out-of-line data and any handles.
func (c *compiler) foreignShape(id fidlgen.EncodedCompoundIdentifier) (outOfLine, handles bool) {
	if info, ok := c.decls[id]; ok && info.TypeShapeV2 != nil {
		return info.TypeShapeV2.MaxOutOfLine > 0, info.TypeShapeV2.MaxHandles > 0
	}
	// Unknown: assume both, the referenced structs are always emitted.
	return true, true
}

func (c *compiler) compilePrimitiveSubtype(val fidlgen.PrimitiveSubtype) Type {
	// TODO(https://fxbug.dev/42121485): Syzkaller does not support enum member references.
	// When this changes, we need to remove all special handling such as
	// ignoring specific files in the codegen test, or in the regen script.
	if t, ok := primitiveTypes[val]; ok {
		return Type(t)
	}
	if t, ok := zxPrimitiveTypes[val]; ok {
		return Type(t)
	}
	panic(fmt.Sprintf("unknown primitive type: %v", val))
}

func (c *compiler) compilePrimitiveSubtypeRange(val fidlgen.PrimitiveSubtype, valRange string) Type {
	return Type(fmt.Sprintf("%s[%s]", c.compilePrimitiveSubtype(val), valRange))
}

// syzkallerHandleResources are the handle resources syzkaller's own Fuchsia
// descriptions declare in sys/fuchsia. A subtype missing from them has no
// resource to refer to, and naming it anyway makes the generated description
// fail to compile.
//
// The syscall descriptions this tool generates from the vDSO declare every
// subtype; pass -assume-vdso-resources when they are in use.
var syzkallerHandleResources = map[string]struct{}{
	"zx_bti":       {},
	"zx_chan":      {},
	"zx_fifo":      {},
	"zx_guest":     {},
	"zx_handle":    {},
	"zx_interrupt": {},
	"zx_iommu":     {},
	"zx_job":       {},
	"zx_msi":       {},
	"zx_pager":     {},
	"zx_pmt":       {},
	"zx_port":      {},
	"zx_process":   {},
	"zx_profile":   {},
	"zx_resource":  {},
	"zx_socket":    {},
	"zx_stream":    {},
	"zx_thread":    {},
	"zx_timer":     {},
	"zx_vmar":      {},
	"zx_vmo":       {},
}

func (c *compiler) compileHandleSubtype(val fidlgen.HandleSubtype) Type {
	t, ok := handleSubtypes[val]
	if !ok {
		panic(fmt.Sprintf("unknown handle type: %v", val))
	}
	if _, declared := syzkallerHandleResources[t]; !declared && !c.opts.AssumeVdsoResources {
		return Type(handleSubtypes[fidlgen.HandleSubtypeNone])
	}
	return Type(t)
}

func (c *compiler) compileEnum(val fidlgen.Enum) Enum {
	e := Enum{
		Name: c.compileCompoundIdentifier(val.Name, ""),
		Type: string(c.compilePrimitiveSubtype(val.Type)),
	}
	for _, v := range val.Members {
		e.Members = append(e.Members, fmt.Sprintf("%s_%s", e.Name, v.Name))
	}
	if val.IsFlexible() {
		e.Members = append(e.Members, fmt.Sprintf("%s__UNKNOWN", e.Name))
	}
	return e
}

func (c *compiler) compileBits(val fidlgen.Bits) Bits {
	e := Bits{
		c.compileCompoundIdentifier(val.Name, ""),
		string(c.compilePrimitiveSubtype(val.Type.PrimitiveSubtype)),
		[]string{},
	}
	for _, v := range val.Members {
		e.Members = append(e.Members, fmt.Sprintf("%s_%s", e.Name, v.Name))
	}
	if val.IsFlexible() {
		e.Members = append(e.Members, fmt.Sprintf("%s__UNKNOWN", e.Name))
	}
	return e
}

func (c *compiler) compileStructMember(p fidlgen.StructMember) (StructMember, *StructMember, *StructMember) {
	return c.compileStructMemberFromNameAndType(p.Name, p.Type)
}

func (c *compiler) compileStructMemberFromNameAndType(name fidlgen.Identifier, typ fidlgen.Type) (StructMember, *StructMember, *StructMember) {
	var i StructMember
	var o *StructMember
	var h *StructMember

	switch typ.Kind {
	case fidlgen.PrimitiveType:
		i = StructMember{
			Type: c.compilePrimitiveSubtype(typ.PrimitiveSubtype),
			Name: c.compileIdentifier(name, ""),
		}
	case fidlgen.HandleType:
		i = StructMember{
			Type: Type("flags[fidl_handle_presence, int32]"),
			Name: c.compileIdentifier(name, ""),
		}

		// Out-of-line handles
		h = &StructMember{
			Type: c.compileHandleSubtype(typ.HandleSubtype),
			Name: c.compileIdentifier(name, ""),
		}
	case fidlgen.EndpointType:
		i = StructMember{
			Type: Type("flags[fidl_handle_presence, int32]"),
			Name: c.compileIdentifier(name, ""),
		}

		// Out-of-line handles
		h = &StructMember{
			Type: Type(fmt.Sprintf("zx_chan_%s_%s", c.compileCompoundIdentifier(typ.Protocol, ""), typ.Role)),
			Name: c.compileIdentifier(name, ""),
		}
	case fidlgen.ArrayType:
		inLine, outOfLine, handle := c.compileStructMemberFromNameAndType(
			fidlgen.Identifier(c.compileIdentifier(name, OutOfLineSuffix)),
			(*typ.ElementType),
		)

		i = StructMember{
			Type: Type(fmt.Sprintf("array[%s, %v]", inLine.Type, *typ.ElementCount)),
			Name: c.compileIdentifier(name, InLineSuffix),
		}

		// Variable-size, out-of-line data
		if outOfLine != nil {
			o = &StructMember{
				Type: Type(fmt.Sprintf("array[%s, %v]", outOfLine.Type, *typ.ElementCount)),
				Name: c.compileIdentifier(name, OutOfLineSuffix),
			}
		}

		// Out-of-line handles
		if handle != nil {
			h = &StructMember{
				Type: Type(fmt.Sprintf("array[%s, %v]", handle.Type, *typ.ElementCount)),
				Name: c.compileIdentifier(name, HandlesSuffix),
			}
		}
	case fidlgen.StringType:
		// Constant-size, in-line data
		i = StructMember{
			Type: Type("fidl_string"),
			Name: c.compileIdentifier(name, InLineSuffix),
		}

		// Variable-size, out-of-line data
		o = &StructMember{
			Type: Type("fidl_aligned[stringnoz]"),
			Name: c.compileIdentifier(name, OutOfLineSuffix),
		}
	case fidlgen.VectorType:
		// Constant-size, in-line data
		i = StructMember{
			Type: Type("fidl_vector"),
			Name: c.compileIdentifier(name, InLineSuffix),
		}

		// Variable-size, out-of-line data
		inLine, outOfLine, handle := c.compileStructMemberFromNameAndType(
			fidlgen.Identifier(c.compileIdentifier(name, OutOfLineSuffix)),
			(*typ.ElementType),
		)
		o = &StructMember{
			Type: Type(fmt.Sprintf("array[%s]", inLine.Type)),
			Name: c.compileIdentifier(name, OutOfLineSuffix),
		}

		if outOfLine != nil {
			o = &StructMember{
				Type: Type(fmt.Sprintf("parallel_array[%s, %s]", inLine.Type, outOfLine.Type)),
				Name: c.compileIdentifier(name, OutOfLineSuffix),
			}
		}

		// Out-of-line handles
		if handle != nil {
			h = &StructMember{
				Type: Type(fmt.Sprintf("array[%s]", handle.Type)),
				Name: c.compileIdentifier(name, ""),
			}
		}
	case fidlgen.InternalType:
		switch typ.InternalSubtype {
		case fidlgen.FrameworkErr:
			// The framework error of a flexible method travels as an int32.
			i = StructMember{
				Type: c.compilePrimitiveSubtype(fidlgen.Int32),
				Name: c.compileIdentifier(name, ""),
			}
		default:
			panic(fmt.Sprintf("unknown internal subtype: %v", typ.InternalSubtype))
		}
	case fidlgen.IdentifierType:
		declInfo, ok := c.decls[typ.Identifier]
		if !ok {
			panic(fmt.Sprintf("unknown identifier: %v", typ.Identifier))
		}

		switch declInfo.Type {
		case fidlgen.EnumDeclType:
			under := c.foreignIntType(typ.Identifier)
			if e, ok := c.enums[typ.Identifier]; ok {
				under = c.compilePrimitiveSubtype(e.Type)
			}
			i = StructMember{
				Type: Type(fmt.Sprintf("flags[%s, %s]", c.compileCompoundIdentifier(typ.Identifier, ""), under)),
				Name: c.compileIdentifier(name, ""),
			}
		case fidlgen.BitsDeclType:
			under := c.foreignIntType(typ.Identifier)
			if b, ok := c.bits[typ.Identifier]; ok {
				under = c.compilePrimitiveSubtype(b.Type.PrimitiveSubtype)
			}
			i = StructMember{
				Type: Type(fmt.Sprintf("flags[%s, %s]", c.compileCompoundIdentifier(typ.Identifier, ""), under)),
				Name: c.compileIdentifier(name, ""),
			}
		case fidlgen.UnionDeclType:
			var hasOutOfLine, hasHandles bool
			u, known := c.unions[typ.Identifier]
			_, onStack := c.compiling[typ.Identifier]
			if known && !onStack {
				c.compiling[typ.Identifier] = struct{}{}
				_, outOfLine, handles := c.compileUnion(u)
				delete(c.compiling, typ.Identifier)
				hasOutOfLine, hasHandles = outOfLine != nil, handles != nil
			} else {
				hasOutOfLine, hasHandles = c.foreignShape(typ.Identifier)
			}

			// Constant-size, in-line data
			t := c.compileCompoundIdentifier(typ.Identifier, InLineSuffix)
			i = StructMember{
				Type: Type(t),
				Name: c.compileIdentifier(name, InLineSuffix),
			}

			// Variable-size, out-of-line data
			if hasOutOfLine {
				t := c.compileCompoundIdentifier(typ.Identifier, OutOfLineSuffix)
				o = &StructMember{
					Type: Type(t),
					Name: c.compileIdentifier(name, OutOfLineSuffix),
				}
			}

			// Out-of-line handles
			if hasHandles {
				t := c.compileCompoundIdentifier(typ.Identifier, HandlesSuffix)
				h = &StructMember{
					Type: Type(t),
					Name: c.compileIdentifier(name, ""),
				}
			}
		case fidlgen.TableDeclType:
			// Constant-size, in-line data: a table is sent as a vector of
			// envelopes, one per ordinal.
			i = StructMember{
				Type: Type("fidl_vector"),
				Name: c.compileIdentifier(name, InLineSuffix),
			}

			// Variable-size, out-of-line data: the envelopes themselves,
			// followed by the values too large to sit inside one.
			o = &StructMember{
				Type: Type(c.compileCompoundIdentifier(typ.Identifier, OutOfLineSuffix)),
				Name: c.compileIdentifier(name, OutOfLineSuffix),
			}

			// Out-of-line handles.
			h = &StructMember{
				Type: Type(c.compileCompoundIdentifier(typ.Identifier, HandlesSuffix)),
				Name: c.compileIdentifier(name, ""),
			}
		case fidlgen.StructDeclType:
			// Fixed-size, in-line data.
			i = StructMember{
				Type: Type(c.compileCompoundIdentifier(typ.Identifier, InLineSuffix)),
				Name: c.compileIdentifier(name, InLineSuffix),
			}

			// Out-of-line data.
			o = &StructMember{
				Type: Type(c.compileCompoundIdentifier(typ.Identifier, OutOfLineSuffix)),
				Name: c.compileIdentifier(name, OutOfLineSuffix),
			}

			// Handles.
			h = &StructMember{
				Type: Type(c.compileCompoundIdentifier(typ.Identifier, HandlesSuffix)),
				Name: c.compileIdentifier(name, ""),
			}
		}
	}

	return i, o, h
}

func header(ordinal uint64) []StructMember {
	return []StructMember{
		{
			Type: Type(fmt.Sprintf("fidl_message_header[%d]", ordinal)),
			Name: "hdr",
		},
	}
}

type members []StructMember

func (members members) voidIfEmpty() members {
	if len(members) == 0 {
		return []StructMember{
			{Name: "void", Type: "void"},
		}
	}
	return members
}

func (members members) uint8PaddingIfEmpty() members {
	if len(members) == 0 {
		return []StructMember{
			{Name: "padding", Type: Type(primitiveTypes[fidlgen.Uint8])},
		}
	}
	return members
}

type result struct {
	Inline, OutOfLine, Handles members
}

func (c *compiler) compileStruct(p fidlgen.Struct) result {
	var result result
	for _, m := range p.Members {
		inLine, outOfLine, handles := c.compileStructMember(m)
		result.Inline = append(result.Inline, inLine)
		if outOfLine != nil {
			result.OutOfLine = append(result.OutOfLine, *outOfLine)
		}
		if handles != nil {
			result.Handles = append(result.Handles, *handles)
		}
	}
	return result
}

func (c *compiler) compileUnion(p fidlgen.Union) ([]StructMember, []StructMember, []StructMember) {
	var i, o, h []StructMember

	for _, m := range p.Members {
		inLine, outOfLine, handles := c.compileStructMemberFromNameAndType(m.Name, m.Type)

		i = append(i, StructMember{
			Type: Type(fmt.Sprintf("fidl_union_member[%d, %s]", m.Ordinal, inLine.Type)),
			Name: inLine.Name,
		})

		if outOfLine != nil {
			o = append(o, *outOfLine)
		}

		if handles != nil {
			h = append(h, *handles)
		}
	}

	return i, o, h
}

// compileTable returns the out-of-line and handle members of a FIDL table.
//
// A table is encoded as a vector of envelopes: `fidl_vector` in-line, then one
// eight-byte envelope per ordinal out-of-line, in ordinal order, with a zeroed
// envelope standing in for every ordinal the table leaves undefined. A value
// of at most four bytes travels inside its own envelope; a larger one follows
// the envelope array, aligned to eight bytes and in ordinal order, with the
// envelope describing its size.
func (c *compiler) compileTable(p fidlgen.Table) ([]StructMember, []StructMember) {
	var envelopes, values, handles []StructMember

	byOrdinal := make(map[int]fidlgen.TableMember, len(p.Members))
	maxOrdinal := 0
	for _, m := range p.Members {
		byOrdinal[m.Ordinal] = m
		if m.Ordinal > maxOrdinal {
			maxOrdinal = m.Ordinal
		}
	}

	for ordinal := 1; ordinal <= maxOrdinal; ordinal++ {
		m, ok := byOrdinal[ordinal]
		if !ok {
			// An ordinal the table does not define is sent as a zeroed envelope.
			envelopes = append(envelopes, StructMember{
				Name: fmt.Sprintf("reserved%d", ordinal),
				Type: Type(fmt.Sprintf("array[const[0, int8], %d]", fidlEnvelopeSize)),
			})
			continue
		}

		inLine, outOfLine, handle := c.compileStructMemberFromNameAndType(m.Name, m.Type)
		inlined := m.Type.TypeShapeV2.InlineSize <= fidlEnvelopeInlineSize
		if inlined {
			envelopes = append(envelopes, StructMember{
				Name: c.compileIdentifier(m.Name, EnvelopeSuffix),
				Type: inLine.Type,
			})
			if padding := fidlEnvelopeInlineSize - m.Type.TypeShapeV2.InlineSize; padding > 0 {
				envelopes = append(envelopes, StructMember{
					Name: c.compileIdentifier(m.Name, EnvelopeSuffix+"Padding"),
					Type: Type(fmt.Sprintf("array[const[0, int8], %d]", padding)),
				})
			}
		} else {
			envelopes = append(envelopes, StructMember{
				Name: c.compileIdentifier(m.Name, EnvelopeSuffix+"NumBytes"),
				Type: Type(primitiveTypes[fidlgen.Uint32]),
			})
			values = append(values, StructMember{
				Name: c.compileIdentifier(m.Name, InLineSuffix),
				Type: Type(fmt.Sprintf("fidl_aligned[%s]", inLine.Type)),
			})
		}
		envelopes = append(envelopes, StructMember{
			Name: c.compileIdentifier(m.Name, EnvelopeSuffix+"NumHandles"),
			Type: Type(primitiveTypes[fidlgen.Uint16]),
		}, StructMember{
			Name: c.compileIdentifier(m.Name, EnvelopeSuffix+"Flags"),
			Type: Type(fmt.Sprintf("const[%d, %s]", boolToInt(inlined), primitiveTypes[fidlgen.Uint16])),
		})

		if outOfLine != nil {
			values = append(values, *outOfLine)
		}
		if handle != nil {
			handles = append(handles, *handle)
		}
	}

	return append(envelopes, values...), handles
}

// boolToInt renders the envelope flag that says whether a value was inlined.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (c *compiler) compileParameters(name string, ordinal uint64, payload *fidlgen.Struct) (Struct, Struct) {
	result := c.compileStruct(fidlgen.Struct{})
	if payload != nil {
		result = c.compileStruct(*payload)
	}
	return Struct{
			Name:    name,
			Members: append(append(header(ordinal), result.Inline...), result.OutOfLine...),
		}, Struct{
			Name:    name + HandlesSuffix,
			Members: result.Handles.voidIfEmpty(),
		}
}

func (c *compiler) compileMethod(protocolName fidlgen.EncodedCompoundIdentifier, val fidlgen.Method) Method {
	methodName := c.compileCompoundIdentifier(protocolName, string(val.Name))
	r := Method{
		Name:    methodName,
		Ordinal: val.Ordinal,
	}

	if val.HasRequest {
		var payload *fidlgen.Struct
		if payloadID, ok := val.GetRequestPayloadIdentifier(); ok {
			if s, ok := c.structs[payloadID]; ok {
				payload = &s
			}
		}
		request, requestHandles := c.compileParameters(r.Name+RequestSuffix, r.Ordinal, payload)
		r.Request = &request
		r.RequestHandles = &requestHandles
		r.Structs = append(r.Structs, request, requestHandles)
	}

	// For response, we only extract handles for now.
	if val.HasResponse {
		var payload *fidlgen.Struct
		if payloadID, ok := val.GetResponsePayloadIdentifier(); ok {
			if s, ok := c.structs[payloadID]; ok {
				payload = &s
			}
		}
		suffix := ResponseSuffix
		if !val.HasRequest {
			suffix = EventSuffix
		}
		response, responseHandles := c.compileParameters(r.Name+suffix, r.Ordinal, payload)
		r.Response = &response
		r.ResponseHandles = &responseHandles
		r.Structs = append(r.Structs, response, responseHandles)
	}

	return r
}

func (c *compiler) compileProtocol(val fidlgen.Protocol) Protocol {
	r := Protocol{
		Name:               c.compileCompoundIdentifier(val.Name, ""),
		ProtocolNameString: strings.Trim(val.GetProtocolName(), "\""),
	}
	for _, v := range val.Methods {
		r.Methods = append(r.Methods, c.compileMethod(val.Name, v))
	}
	return r
}

func compile(fidlData fidlgen.Root, opts Options) Root {
	fidlData = fidlData.ForBindings("syzkaller")
	root := Root{
		Experiments: fidlData.Experiments,
	}
	libraryName := fidlData.Name.Parse()
	c := compiler{
		decls:     fidlData.DeclInfo(),
		structs:   make(StructMap),
		tables:    make(TableMap),
		unions:    make(UnionMap),
		enums:     make(EnumMap),
		bits:      make(BitsMap),
		compiling: make(map[fidlgen.EncodedCompoundIdentifier]struct{}),
		library:   libraryName,
		opts:      opts,
	}

	root.HeaderPath = fmt.Sprintf("%s/c/fidl.h", formatLibraryPath(libraryName))

	// syzkaller keeps structs and unions in one namespace and rejects a name
	// declared twice. The same declaration is reached by more than one route
	// here: an anonymous method payload, for instance, is compiled both as a
	// top-level layout and as the payload of the method that names it.
	declared := make(map[string]struct{})
	addStruct := func(s Struct) {
		if _, ok := declared[s.Name]; ok {
			return
		}
		declared[s.Name] = struct{}{}
		root.Structs = append(root.Structs, s)
	}
	addUnion := func(u Union) {
		if _, ok := declared[u.Name]; ok {
			return
		}
		declared[u.Name] = struct{}{}
		root.Unions = append(root.Unions, u)
	}

	// Register every declaration before compiling any of them: a struct or a
	// table can name a union that the JSON IR lists after it, and looking the
	// union up too early silently drops its out-of-line and handle members.
	for _, v := range fidlData.Enums {
		c.enums[v.Name] = v
	}
	for _, v := range fidlData.Bits {
		c.bits[v.Name] = v
	}
	for _, v := range fidlData.Structs {
		c.structs[v.Name] = v
	}
	for _, v := range fidlData.ExternalStructs {
		c.structs[v.Name] = v
	}
	for _, v := range fidlData.Tables {
		c.tables[v.Name] = v
	}
	for _, v := range fidlData.Unions {
		c.unions[v.Name] = v
	}

	for _, v := range fidlData.Enums {
		root.Enums = append(root.Enums, c.compileEnum(v))
	}

	for _, v := range fidlData.Bits {
		root.Bits = append(root.Bits, c.compileBits(v))
	}

	for _, v := range fidlData.Structs {
		result := c.compileStruct(v)
		addStruct(Struct{
			Name:    c.compileCompoundIdentifier(v.Name, InLineSuffix),
			Members: result.Inline.uint8PaddingIfEmpty(),
		})

		addStruct(Struct{
			Name:    c.compileCompoundIdentifier(v.Name, OutOfLineSuffix),
			Members: result.OutOfLine.voidIfEmpty(),
		})

		addStruct(Struct{
			Name:    c.compileCompoundIdentifier(v.Name, HandlesSuffix),
			Members: result.Handles.voidIfEmpty(),
		})
	}

	for _, v := range fidlData.Tables {
		outOfLine, handles := c.compileTable(v)
		addStruct(Struct{
			Name:    c.compileCompoundIdentifier(v.Name, OutOfLineSuffix),
			Members: members(outOfLine).voidIfEmpty(),
		})

		addStruct(Struct{
			Name:    c.compileCompoundIdentifier(v.Name, HandlesSuffix),
			Members: members(handles).voidIfEmpty(),
		})
	}

	for _, v := range fidlData.Unions {
		i, o, h := c.compileUnion(v)
		addUnion(Union{
			Name: c.compileCompoundIdentifier(v.Name, InLineSuffix),
			// An empty union has no variant to select, but syzkaller still
			// needs a field to lay out.
			Members: members(i).uint8PaddingIfEmpty(),
		})

		if len(o) == 0 {
			o = append(o, StructMember{
				Name: "void",
				Type: "void",
			})
		}

		if len(h) == 0 {
			h = append(h, StructMember{
				Name: "void",
				Type: "void",
			})
		}

		addUnion(Union{
			Name:    c.compileCompoundIdentifier(v.Name, OutOfLineSuffix),
			Members: o,
			VarLen:  true,
		})

		addUnion(Union{
			Name:    c.compileCompoundIdentifier(v.Name, HandlesSuffix),
			Members: h,
			VarLen:  true,
		})
	}

	for _, v := range fidlData.Protocols {
		root.Protocols = append(root.Protocols, c.compileProtocol(v))
	}

	for _, i := range root.Protocols {
		for _, m := range i.Methods {
			for _, s := range m.Structs {
				addStruct(s)
			}
			for _, s := range m.Unions {
				addUnion(s)
			}
		}
	}

	return root
}
