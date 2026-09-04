// Copyright 2022 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fidlgen

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type PaddingMarker struct {
	// Offset into the struct in bytes (0 is the start of the struct).
	Offset int
	// Width of the mask in bits. Either 16, 32, or 64.
	MaskBitWidth int
	// Little endian mask with 0x00 for non-padding and 0xff for padding.
	// Only the lower MaskBitWidth bits are used. The higher bits are 0.
	Mask uint64
}

type PaddingConfig struct {
	FlattenStructs bool
	// Only allowed when FlattenStructs is true.
	FlattenArrays bool
	// Must be provided if FlattenStructs is true.
	ResolveStruct func(identifier EncodedCompoundIdentifier) *Struct
}

func (s Struct) BuildPaddingMarkers(config PaddingConfig) []PaddingMarker {
	if config.FlattenArrays && !config.FlattenStructs {
		panic("flattening arrays but not structs is pointless: arrays of non-structs have no padding")
	}

	var paddingMarkers []PaddingMarker

	// Construct a mask across the whole struct with 0xff bytes where there is padding.
	fullStructMask := make([]byte, s.TypeShapeV2.InlineSize)
	populateFullStructMaskForStruct(fullStructMask, config, s)

	// Split up the mask into aligned integer mask segments.
	// Only the sections needing padding are outputted.
	// e.g. 00 00 00 00 00 00 00 00 00 ff ff 00 ff ff ff ff
	//   -> 0000000000000000, 00ffff00ffffffff
	//   -> []PaddingMarker{{Offset: 8, MaskBitWidth: 64, Mask: binary.LittleEndian.Uint64([]byte{0x00, 0xff, 0xff, 0x00, 0xff, 0xff, 0xff, 0xff})}}
	//   -> []PaddingMarker{{Offset: 8, MaskBitWidth: 64, Mask: 0xff_ff_ff_ff_00_ff_ff_00)}}
	extractNonzeroSliceOffsets := func(stride int) []int {
		var offsets []int
		for endi := stride - 1; endi < len(fullStructMask); endi += stride {
			i := endi - (stride - 1)
			if bytes.Contains(fullStructMask[i:i+stride], []byte{0xff}) {
				offsets = append(offsets, i)
			}
		}
		return offsets
	}
	littleEndian := func(b []byte, size int) uint64 {
		switch size {
		case 8:
			return binary.LittleEndian.Uint64(b)
		case 4:
			return uint64(binary.LittleEndian.Uint32(b))
		case 2:
			return uint64(binary.LittleEndian.Uint16(b))
		default:
			panic("unexpected size")
		}
	}
	for _, size := range []int{8, 4, 2} {
		if s.TypeShapeV2.Alignment < size {
			continue
		}
		for _, i := range extractNonzeroSliceOffsets(size) {
			s := fullStructMask[i : i+size]
			paddingMarkers = append(paddingMarkers, PaddingMarker{
				Offset:       i,
				MaskBitWidth: size * 8,
				Mask:         littleEndian(s, size),
			})
			// Zero out the region to avoid including it in smaller masks.
			for j := range s {
				s[j] = 0
			}
		}
	}
	if bytes.Contains(fullStructMask, []byte{0xff}) {
		// This shouldn't be possible because it requires an alignment 1 struct to have padding.
		panic(fmt.Sprintf("expected mask to be zero, was %v", fullStructMask))
	}
	return paddingMarkers
}

func populateFullStructMaskForStruct(mask []byte, config PaddingConfig, s Struct) {
	paddingEnd := s.TypeShapeV2.InlineSize - 1
	for i := len(s.Members) - 1; i >= 0; i-- {
		member := s.Members[i]
		fieldShape := member.FieldShapeV2
		populateFullStructMaskForType(mask[fieldShape.Offset:paddingEnd+1], config, &member.Type)
		for j := 0; j < fieldShape.Padding; j++ {
			mask[paddingEnd-j] = 0xff
		}
		paddingEnd = fieldShape.Offset - 1
	}
}

func populateFullStructMaskForType(mask []byte, config PaddingConfig, typ *Type) {
	if typ.Nullable {
		return
	}
	switch typ.Kind {
	case ArrayType:
		if config.FlattenArrays {
			elemByteSize := len(mask) / *typ.ElementCount
			for i := 0; i < *typ.ElementCount; i++ {
				populateFullStructMaskForType(mask[i*elemByteSize:(i+1)*elemByteSize], config, typ.ElementType)
			}
		}
	case IdentifierType:
		if config.FlattenStructs {
			if s := config.ResolveStruct(typ.Identifier); s != nil {
				populateFullStructMaskForStruct(mask, config, *s)
			}
		}
	}
}
