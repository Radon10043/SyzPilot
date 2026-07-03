package fidl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a trimmed but structurally faithful FIDL IR fixture covering an enum, a struct
// that references the enum and a handle, and a protocol.
const sampleIR = `{
  "name": "fuchsia.hardware.scsi",
  "enum_declarations": [
    {
      "name": "fuchsia.hardware.scsi/ReadBufferMode",
      "type": "uint8",
      "members": [
        {"name": "VENDOR_SPECIFIC", "value": {"value": "1", "expression": "0x01"}},
        {"name": "DATA", "value": {"value": "2", "expression": "0x02"}}
      ]
    }
  ],
  "struct_declarations": [
    {
      "name": "fuchsia.hardware.scsi/ScsiReadBufferRequest",
      "members": [
        {"name": "lun", "type": {"kind_v2": "primitive", "subtype": "uint16"}},
        {"name": "mode", "type": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.scsi/ReadBufferMode", "nullable": false}},
        {"name": "data", "type": {"kind_v2": "handle", "subtype": "vmo", "nullable": false}}
      ]
    }
  ],
  "protocol_declarations": [
    {
      "name": "fuchsia.hardware.scsi/Scsi",
      "methods": [
        {
          "name": "ReadBuffer",
          "has_request": true,
          "maybe_request_payload": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.scsi/ScsiReadBufferRequest"},
          "has_response": true,
          "maybe_response_payload": {"kind_v2": "identifier", "identifier": "fuchsia.hardware.scsi/Scsi_ReadBuffer_Result"}
        }
      ]
    }
  ]
}`

func loadFixture(t *testing.T) *Index {
	t.Helper()
	dir := t.TempDir()
	// mirror the real layout: <dir>/<library>/<library>.fidl.json
	libDir := filepath.Join(dir, "fuchsia.hardware.scsi")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "fuchsia.hardware.scsi.fidl.json"), []byte(sampleIR), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	// a deeper *.fidl.json must be ignored by Load
	deepDir := filepath.Join(libDir, "nested")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("mkdir deep fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deepDir, "ignored.fidl.json"), []byte(`{"name":"should.not.load","enum_declarations":[{"name":"should.not.load/Ignored","type":"uint8","members":[]}]}`), 0644); err != nil {
		t.Fatalf("write deep fixture: %v", err)
	}
	// same glob convention as the production caller in generator-fuchsia.go
	idx, err := Load(filepath.Join(dir, "*", "*.fidl.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return idx
}

func TestGetResolvesNamesAndSuffixes(t *testing.T) {
	idx := loadFixture(t)

	cases := []struct {
		query string
		want  []string // substrings that must appear in the rendering
	}{
		{"fuchsia.hardware.scsi/ReadBufferMode", []string{"enum", "VENDOR_SPECIFIC = 0x01", "DATA = 0x02"}},
		{"fuchsia_hardware_scsi_ReadBufferMode", []string{"enum"}},
		// syzlang node name with a generated suffix must resolve to the base struct
		{"fuchsia_hardware_scsi_ScsiReadBufferRequestInLine", []string{"struct", "lun uint16", "mode fuchsia.hardware.scsi/ReadBufferMode", "data handle<vmo>"}},
		{"fuchsia_hardware_scsi_ScsiReadBufferRequestHandles", []string{"struct"}},
		{"fuchsia_hardware_scsi_Scsi", []string{"protocol", "ReadBuffer(fuchsia.hardware.scsi/ScsiReadBufferRequest) -> (fuchsia.hardware.scsi/Scsi_ReadBuffer_Result)"}},
	}
	for _, c := range cases {
		got, ok := idx.Get(c.query)
		if !ok {
			t.Errorf("Get(%q): not found", c.query)
			continue
		}
		for _, want := range c.want {
			if !strings.Contains(got, want) {
				t.Errorf("Get(%q): missing %q in:\n%s", c.query, want, got)
			}
		}
	}
}

func TestGetUnknownReturnsFalse(t *testing.T) {
	idx := loadFixture(t)
	if _, ok := idx.Get("fuchsia_does_not_Exist"); ok {
		t.Errorf("Get of unknown name should return false")
	}
}

func TestLoadIgnoresDeeperFiles(t *testing.T) {
	idx := loadFixture(t)
	// the *.fidl.json two levels deep must not be indexed
	if _, ok := idx.Get("should.not.load/Ignored"); ok {
		t.Errorf("Load should ignore *.fidl.json deeper than one library level")
	}
}

// an IR fixture carrying type_shape_v2 / field_shape_v2 so we can assert the
// rendered declaration is annotated with wire geometry.
const shapedIR = `{
  "name": "fuchsia.overnet.protocol",
  "struct_declarations": [
    {
      "name": "fuchsia.overnet.protocol/ChannelHandle",
      "type_shape_v2": {"inline_size": 24, "alignment": 8, "max_out_of_line": 32, "max_handles": 1},
      "members": [
        {
          "name": "rights",
          "type": {"kind_v2": "primitive", "subtype": "uint32", "type_shape_v2": {"inline_size": 4, "max_out_of_line": 0, "max_handles": 0}},
          "field_shape_v2": {"offset": 0, "padding": 4}
        },
        {
          "name": "stream_ref",
          "type": {"kind_v2": "identifier", "identifier": "fuchsia.overnet.protocol/StreamRef", "type_shape_v2": {"inline_size": 16, "max_out_of_line": 32, "max_handles": 1}},
          "field_shape_v2": {"offset": 8, "padding": 0}
        }
      ]
    }
  ]
}`

func TestRenderIncludesWireShape(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "fuchsia.overnet.protocol")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "fuchsia.overnet.protocol.fidl.json"), []byte(shapedIR), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	idx, err := Load(filepath.Join(dir, "*", "*.fidl.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got, ok := idx.Get("fuchsia_overnet_protocol_ChannelHandle")
	if !ok {
		t.Fatalf("Get: declaration not found")
	}
	for _, want := range []string{
		"// wire_v2: inline_size=24 alignment=8 max_out_of_line=32 max_handles=1",
		"rights uint32\t// @0 size=4 pad=4",
		"stream_ref fuchsia.overnet.protocol/StreamRef\t// @8 size=16 out_of_line<=32 handles<=1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("render missing %q in:\n%s", want, got)
		}
	}
}

func TestSearch(t *testing.T) {
	idx := loadFixture(t)

	res := idx.Search("ReadBuffer", 10)
	if len(res) == 0 {
		t.Fatalf("Search(ReadBuffer): expected results, got none")
	}
	joined := strings.Join(res, "\n")
	for _, want := range []string{
		"fuchsia_hardware_scsi_ReadBufferMode (enum)",
		"fuchsia_hardware_scsi_ScsiReadBufferRequest (struct)",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("Search(ReadBuffer): missing %q in:\n%s", want, joined)
		}
	}

	if got := idx.Search("nonexistent-keyword", 10); got != nil {
		t.Errorf("Search of unknown keyword: expected nil, got %v", got)
	}
}
