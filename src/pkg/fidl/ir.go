// Package fidl loads FIDL IR JSON (*.fidl.json) files and exposes a small query
// API used by the spec-fixing agent to consult the original FIDL definitions
// behind the generated syzlang specs.
package fidl

// root is the minimal subset of the FIDL IR JSON schema we care about. Fields we
// do not need (type shapes, locations, attributes, ...) are intentionally omitted
// and ignored during unmarshalling.
type root struct {
	Name                       string         `json:"name"`
	ConstDeclarations          []constDecl    `json:"const_declarations"`
	EnumDeclarations           []enumDecl     `json:"enum_declarations"`
	BitsDeclarations           []bitsDecl     `json:"bits_declarations"`
	StructDeclarations         []recordDecl   `json:"struct_declarations"`
	ExternalStructDeclarations []recordDecl   `json:"external_struct_declarations"`
	UnionDeclarations          []recordDecl   `json:"union_declarations"`
	TableDeclarations          []recordDecl   `json:"table_declarations"`
	ProtocolDeclarations       []protocolDecl `json:"protocol_declarations"`
	AliasDeclarations          []aliasDecl    `json:"alias_declarations"`
	NewTypeDeclarations        []aliasDecl    `json:"new_type_declarations"`
}

// fidlType mirrors a FIDL "type" object (kind_v2 based). A single struct covers
// every kind; only the fields relevant to that kind are populated.
type fidlType struct {
	Kind              string     `json:"kind_v2"`
	Subtype           string     `json:"subtype"`             // primitive / handle / internal
	Identifier        string     `json:"identifier"`          // identifier
	Nullable          bool       `json:"nullable"`            // identifier / string / vector
	ElementType       *fidlType  `json:"element_type"`        // vector / array
	ElementCount      *int       `json:"element_count"`       // array
	MaybeElementCount *int       `json:"maybe_element_count"` // string / vector
	Role              string     `json:"role"`                // endpoint: client / server
	Protocol          string     `json:"protocol"`            // endpoint
	TypeShape         *typeShape `json:"type_shape_v2"`       // wire geometry of this type
}

// typeShape mirrors the FIDL "type_shape_v2" object: the wire-format geometry of a
// type. Exposing it lets the spec-fixing agent decide the InLine/OutOfLine/Handles
// split (does a member carry out-of-line data? does it carry handles?) and lay out
// inter-field padding exactly, instead of guessing.
type typeShape struct {
	InlineSize          int  `json:"inline_size"`
	Alignment           int  `json:"alignment"`
	Depth               int  `json:"depth"`
	MaxHandles          int  `json:"max_handles"`
	MaxOutOfLine        int  `json:"max_out_of_line"`
	HasPadding          bool `json:"has_padding"`
	HasFlexibleEnvelope bool `json:"has_flexible_envelope"`
}

// fieldShape mirrors the FIDL "field_shape_v2" object attached to a struct member:
// its byte offset within the inline struct and the trailing padding after it. The
// padding count is exactly what a syzlang `padding array[const[0, int8], N]` filler
// must span.
type fieldShape struct {
	Offset  int `json:"offset"`
	Padding int `json:"padding"`
}

// constValue is the value object attached to consts and enum/bits members.
type constValue struct {
	Value      string `json:"value"`      // resolved decimal/string value
	Expression string `json:"expression"` // original expression, e.g. "0x01"
}

// member is shared by record fields (struct/union/table) and enum/bits members.
type member struct {
	Name       string      `json:"name"`
	Type       *fidlType   `json:"type"`           // record members
	Ordinal    int         `json:"ordinal"`        // union / table members
	Value      *constValue `json:"value"`          // enum / bits members
	FieldShape *fieldShape `json:"field_shape_v2"` // struct members: offset + trailing padding
}

type constDecl struct {
	Name  string      `json:"name"`
	Type  *fidlType   `json:"type"`
	Value *constValue `json:"value"`
}

type enumDecl struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"` // underlying integer subtype, e.g. "uint8"
	Members []member `json:"members"`
}

type bitsDecl struct {
	Name    string    `json:"name"`
	Type    *fidlType `json:"type"`
	Mask    string    `json:"mask"`
	Members []member  `json:"members"`
}

// recordDecl covers structs, unions, and tables, which share the same shape for
// our purposes (a name plus a list of members, plus the declaration's wire geometry).
type recordDecl struct {
	Name      string     `json:"name"`
	Members   []member   `json:"members"`
	TypeShape *typeShape `json:"type_shape_v2"`
}

type aliasDecl struct {
	Name string    `json:"name"`
	Type *fidlType `json:"type"`
}

type method struct {
	Name                 string    `json:"name"`
	HasRequest           bool      `json:"has_request"`
	MaybeRequestPayload  *fidlType `json:"maybe_request_payload"`
	HasResponse          bool      `json:"has_response"`
	MaybeResponsePayload *fidlType `json:"maybe_response_payload"`
	HasError             bool      `json:"has_error"`
}

type protocolDecl struct {
	Name    string   `json:"name"`
	Methods []method `json:"methods"`
}
