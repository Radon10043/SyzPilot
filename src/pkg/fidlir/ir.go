package fidlir

import (
	"encoding/json"
)

type Fidlir struct {
	Name          string         `json:"name"`
	BitsDecls     []BitsDecl     `json:"bits_declarations"`
	ConstDecls    []ConstDecl    `json:"const_declarations"`
	EnumDecls     []EnumDecl     `json:"enum_declarations"`
	ProtocolDecls []ProtocolDecl `json:"protocol_declarations"`
	ServiceDecls  []ServiceDecl  `json:"service_declarations"`
	StructDecls   []StructDecl   `json:"struct_declarations"`
	TableDecls    []TableDecl    `json:"table_declarations"`
	UnionDecls    []UnionDecl    `json:"union_declarations"`
	AliasDecls    []AliasDecl    `json:"alias_declarations"`
	Declarations  DeclMap        `json:"declarations"`
}

// FidlType mirrors the "type" object in fidl json, this struct try to cover
// all possible field.
type FidlType struct {
	Kind         string     `json:"kind_v2"`
	Subtype      string     `json:"subtype"`
	Identifier   string     `json:"identifier"`
	Nullable     bool       `json:"nullable"`
	ElemType     *FidlType  `json:"element_type"`
	ElemCount    *int       `json:"element_count"`
	MaybeElemCnt *int       `json:"maybe_element_count"`
	Role         string     `json:"role"`
	Protocol     string     `json:"protocol"`
	TypeShape    *TypeShape `json:"type_shape_v2"`
}

// TypeShape mirrors "type_shape_v2" field in fidl json.
type TypeShape struct {
	InlineSize   int  `json:"inline_size"`
	Alignment    int  `json:"alignment"`
	Depth        int  `json:"depth"`
	MaxHandles   int  `json:"max_handles"`
	MaxOutOfLine int  `json:"max_out_of_line"`
	HasPadding   bool `json:"has_padding"`
	HasFlexEnve  bool `json:"has_flexible_envelope"`
}

// FieldShape mirrors the "field_shape_v2" field.
type FieldShape struct {
	Offset  int `json:"offset"`
	Padding int `json:"padding"`
}

// Member mirrors the "member" object in fidl json, this struct try to cover
// all possible field.
type Member struct {
	Name       string      `json:"name"`
	Type       *FidlType   `json:"type"`
	Ordinal    int         `json:"ordinal"`
	Value      *Value      `json:"value"`
	Attrs      []Attribute `json:"maybe_attributes"`
	FieldShape *FieldShape `json:"field_shape_v2"`
}

// Method mirrors the "method" object in fidl json, which is usually found in ProtocolDecl.Methods.
type Method struct {
	Kind             string          `json:"kind"`
	Ordinal          int             `json:"ordinal"`
	Name             string          `json:"name"`
	HasReq           bool            `json:"has_request"`
	Attrs            []Attribute     `json:"maybe_attributes"`
	MaybeReqPayload  json.RawMessage `json:"maybe_request_payload"`
	HasResp          bool            `json:"has_response"`
	MaybeRespPayload json.RawMessage `json:"maybe_response_payload"`
}

// Literal mirrors the "literal" object in fidl json.
type Literal struct {
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	Expression string `json:"expression"`
}

// Value mirrors the "value" object in fidl json.
type Value struct {
	Kind       string   `json:"kind"`
	Value      string   `json:"value"`
	Expression string   `json:"expression"`
	Literal    *Literal `json:"literal"`
	Identifier string   `json:"identifier"`
}

// Attribute mirrors the member in "maybe_attributes" field in fidl json.
type Attribute struct {
	Name string     `json:"name"`
	Args []Argument `json:"arguments"`
}

// Argument mirrors the member in "arguments" field in fidl json.
type Argument struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value *Value `json:"value"`
}

// ConstDecl mirrors the member in "const_declarations" field in fidl json.
type ConstDecl struct {
	Name  string    `json:"name"`
	Type  *FidlType `json:"type" gorm:"serializer:json;type:text"`
	Value *Value    `json:"value" gorm:"serializer:json;type:text"`
}

// BitsDecl mirrors the member in "bits_declarations" field in fidl json.
type BitsDecl struct {
	Name    string    `json:"name"`
	Type    *FidlType `json:"type" gorm:"serializer:json;type:text"`
	Members []Member  `json:"members" gorm:"serializer:json;type:text"`
}

// EnumDecl mirrors the member in "enum_declarations" field in fidl json.
type EnumDecl struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Members []Member `json:"members" gorm:"serializer:json;type:text"`
}

// ProtocolDecl mirrors the member in "protocol_declarations" field in fidl json.
type ProtocolDecl struct {
	Name    string      `json:"name"`
	Attrs   []Attribute `json:"maybe_attributes" gorm:"serializer:json;type:text"`
	Methods []Method    `json:"methods" gorm:"serializer:json;type:t/btwext"`
}

// ServiceDecl mirrors the member in "service_declarations" field in fidl json.
type ServiceDecl struct {
	Name    string   `json:"name"`
	Members []Member `json:"members" gorm:"serializer:json;type:text"`
}

// StructDecl mirrors the member in "struct_declarations" field in fidl json.
type StructDecl struct {
	Name      string     `json:"name"`
	Members   []Member   `json:"members" gorm:"serializer:json;type:text"`
	TypeShape *TypeShape `json:"type_shape_v2" gorm:"serializer:json;type:text"`
}

// TableDecl mirrors the member in "table_declarations" field in fidl json.
type TableDecl struct {
	Name      string     `json:"name"`
	Members   []Member   `json:"members" gorm:"serializer:json;type:text"`
	TypeShape *TypeShape `json:"type_shape_v2" gorm:"serializer:json;type:text"`
}

// UnionDecl mirrors the member in "union_declarations" field in fidl json.
type UnionDecl struct {
	Name      string     `json:"name"`
	Members   []Member   `json:"members" gorm:"serializer:json;type:text"`
	TypeShape *TypeShape `json:"type_shape_v2" gorm:"serializer:json;type:text"`
}

// AliasDecl mirrors the member in "alias_declarations" field in fidl json.
type AliasDecl struct {
	Name string    `json:"name"`
	Type *FidlType `json:"type" gorm:"serializer:json;type:text"`
}

type DeclMap map[string]string
