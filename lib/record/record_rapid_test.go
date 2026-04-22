// Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package record

import (
	"bytes"
	"testing"

	"pgregory.net/rapid"

	"github.com/openGemini/openGemini/lib/codec"
	"github.com/openGemini/openGemini/lib/util/lifted/vm/protoparser/influx"
)

func generateField(t *rapid.T) *Field {
	fieldType := rapid.SampledFrom([]influx.Field_Type{
		influx.Field_Type_Int,
		influx.Field_Type_Float,
		influx.Field_Type_String,
		influx.Field_Type_Boolean,
	}).Draw(t, "type")

	name := rapid.StringMatching("[a-zA-Z][a-zA-Z0-9_]*").Draw(t, "name")

	return &Field{
		Name: name,
		Type: fieldType,
	}
}

func generateSchema(t *rapid.T, minFields, maxFields int) []*Field {
	numFields := rapid.IntRange(minFields, maxFields).Draw(t, "numFields")
	schema := make([]*Field, numFields)

	seenNames := make(map[string]bool)
	for i := range schema {
		field := generateField(t)
		for seenNames[field.Name] {
			field.Name = rapid.StringMatching("[a-zA-Z][a-zA-Z0-9_]*").Draw(t, "name")
		}
		seenNames[field.Name] = true
		schema[i] = field
	}

	return schema
}

func generateColVal(t *rapid.T, fieldType influx.Field_Type) ColVal {
	colVal := ColVal{}

	numValues := rapid.IntRange(0, 100).Draw(t, "numValues")

	switch fieldType {
	case influx.Field_Type_Int:
		colVal.Init(nil, numValues, numValues)
		for i := 0; i < numValues; i++ {
			if rapid.Bool().Draw(t, "isNull") {
				colVal.AppendNil()
			} else {
				val := rapid.Int64().Draw(t, "value")
				colVal.AppendInteger(val)
			}
		}

	case influx.Field_Type_Float:
		colVal.Init(nil, numValues, numValues)
		for i := 0; i < numValues; i++ {
			if rapid.Bool().Draw(t, "isNull") {
				colVal.AppendNil()
			} else {
				val := rapid.Float64().Draw(t, "value")
				colVal.AppendFloat(val)
			}
		}

	case influx.Field_Type_Boolean:
		colVal.Init(nil, numValues, numValues)
		for i := 0; i < numValues; i++ {
			if rapid.Bool().Draw(t, "isNull") {
				colVal.AppendNil()
			} else {
				val := rapid.Bool().Draw(t, "value")
				colVal.AppendBoolean(val)
			}
		}

	case influx.Field_Type_String:
		colVal.Init(nil, numValues, numValues)
		for i := 0; i < numValues; i++ {
			if rapid.Bool().Draw(t, "isNull") {
				colVal.AppendNil()
			} else {
				val := rapid.String().Draw(t, "value")
				colVal.AppendString(val)
			}
		}
	}

	return colVal
}

func generateRecord(t *rapid.T) *Record {
	schema := generateSchema(t, 0, 20)
	rec := &Record{
		Schema:  schema,
		ColVals: make([]ColVal, len(schema)),
	}

	for i, field := range schema {
		rec.ColVals[i] = generateColVal(t, field.Type)
	}

	return rec
}

func TestRecord_MarshalUnmarshal_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := generateRecord(t)

		var buf []byte
		buf = original.Marshal(buf)

		decoded := &Record{}
		decoded.Unmarshal(buf)

		if len(original.Schema) != len(decoded.Schema) {
			t.Fatalf("Schema length mismatch: got %d, want %d", len(decoded.Schema), len(original.Schema))
		}

		if len(original.ColVals) != len(decoded.ColVals) {
			t.Fatalf("ColVals length mismatch: got %d, want %d", len(decoded.ColVals), len(original.ColVals))
		}

		for i, field := range original.Schema {
			if decoded.Schema[i].Name != field.Name {
				t.Fatalf("Schema[%d].Name mismatch: got %s, want %s", i, decoded.Schema[i].Name, field.Name)
			}
			if decoded.Schema[i].Type != field.Type {
				t.Fatalf("Schema[%d].Type mismatch: got %v, want %v", i, decoded.Schema[i].Type, field.Type)
			}
		}

		for i := range original.ColVals {
			if !colValsEqual(original.ColVals[i], decoded.ColVals[i]) {
				t.Fatalf("ColVals[%d] mismatch", i)
			}
		}
	})
}

func colValsEqual(a, b ColVal) bool {
	if a.Len != b.Len || a.NilCount != b.NilCount {
		return false
	}

	for i := 0; i < a.Len; i++ {
		if a.IsNil(i) != b.IsNil(i) {
			return false
		}

		if !a.IsNil(i) && a.Len > 0 {
			aVal, ok := a.IntegerValue(i)
			bVal, ok2 := b.IntegerValue(i)
			if ok && ok2 && aVal != bVal {
				return false
			}

			aValF, ok := a.FloatValue(i)
			bValF, ok2 := b.FloatValue(i)
			if ok && ok2 && aValF != bValF {
				return false
			}

			aValB, ok := a.BooleanValue(i)
			bValB, ok2 := b.BooleanValue(i)
			if ok && ok2 && aValB != bValB {
				return false
			}

			aValS, ok := a.StringValue(i)
			bValS, ok2 := b.StringValue(i)
			if ok && ok2 && aValS != bValS {
				return false
			}
		}
	}

	return true
}

func TestRecord_EmptyRecord_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := &Record{
			Schema:  []*Field{},
			ColVals: []ColVal{},
		}

		var buf []byte
		buf = original.Marshal(buf)

		decoded := &Record{}
		decoded.Unmarshal(buf)

		if len(decoded.Schema) != 0 || len(decoded.ColVals) != 0 {
			t.Fatalf("Empty record roundtrip failed")
		}
	})
}

func TestRecord_SingleField_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldType := rapid.SampledFrom([]influx.Field_Type{
			influx.Field_Type_Int,
			influx.Field_Type_Float,
			influx.Field_Type_String,
			influx.Field_Type_Boolean,
		}).Draw(t, "type")

		original := &Record{
			Schema: []*Field{
				{
					Name: "test_field",
					Type: fieldType,
				},
			},
			ColVals: []ColVal{
				generateColVal(t, fieldType),
			},
		}

		var buf []byte
		buf = original.Marshal(buf)

		decoded := &Record{}
		decoded.Unmarshal(buf)

		if len(decoded.Schema) != 1 || len(decoded.ColVals) != 1 {
			t.Fatalf("Single field record roundtrip failed")
		}

		if !colValsEqual(original.ColVals[0], decoded.ColVals[0]) {
			t.Fatalf("Single field values mismatch")
		}
	})
}

func TestRecord_LargeRecord_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numFields := rapid.IntRange(50, 100).Draw(t, "numFields")
		original := &Record{
			Schema: generateSchema(t, numFields, numFields),
		}
		original.ColVals = make([]ColVal, numFields)

		for i, field := range original.Schema {
			original.ColVals[i] = generateColVal(t, field.Type)
		}

		var buf []byte
		buf = original.Marshal(buf)

		decoded := &Record{}
		decoded.Unmarshal(buf)

		if len(original.Schema) != len(decoded.Schema) || len(original.ColVals) != len(decoded.ColVals) {
			t.Fatalf("Large record roundtrip failed")
		}

		for i := range original.ColVals {
			if !colValsEqual(original.ColVals[i], decoded.ColVals[i]) {
				t.Fatalf("Large record ColVals[%d] mismatch", i)
			}
		}
	})
}

func TestRecord_WithNulls_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := generateRecord(t)

		for i := range original.ColVals {
			if original.ColVals[i].Len > 0 {
				numNulls := rapid.IntRange(0, original.ColVals[i].Len).Draw(t, "numNulls")
				for j := 0; j < numNulls; j++ {
					pos := rapid.IntRange(0, original.ColVals[i].Len-1).Draw(t, "pos")
					original.ColVals[i].SetNil(pos)
				}
			}
		}

		var buf []byte
		buf = original.Marshal(buf)

		decoded := &Record{}
		decoded.Unmarshal(buf)

		for i := range original.ColVals {
			if original.ColVals[i].NilCount != decoded.ColVals[i].NilCount {
				t.Fatalf("NilCount mismatch for ColVals[%d]: got %d, want %d",
					i, decoded.ColVals[i].NilCount, original.ColVals[i].NilCount)
			}

			for j := 0; j < original.ColVals[i].Len; j++ {
				if original.ColVals[i].IsNil(j) != decoded.ColVals[i].IsNil(j) {
					t.Fatalf("Nil value mismatch at ColVals[%d][%d]", i, j)
				}
			}
		}
	})
}

func TestRecord_CodecSizeAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rec := generateRecord(t)

		expectedSize := rec.CodecSize()
		var buf []byte
		buf = rec.Marshal(buf)
		actualSize := len(buf)

		if actualSize != expectedSize {
			t.Fatalf("CodecSize mismatch: expected %d, got %d", expectedSize, actualSize)
		}
	})
}

func TestRecord_MarshalDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rec := generateRecord(t)

		var buf1, buf2 []byte
		buf1 = rec.Marshal(buf1)
		buf2 = rec.Marshal(buf2)

		if !bytes.Equal(buf1, buf2) {
			t.Fatalf("Marshal should be deterministic")
		}
	})
}

func TestRecord_SpecialCharacters(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		specialChars := rapid.SampledFrom([]string{
			"_test_field",
			"test_field_123",
			"test-field",
			"test.field",
			"test\nfield",
			"test\tfield",
			"test\x00field",
		}).Draw(t, "name")

		fieldType := rapid.SampledFrom([]influx.Field_Type{
			influx.Field_Type_String,
			influx.Field_Type_Int,
		}).Draw(t, "type")

		original := &Record{
			Schema: []*Field{
				{
					Name: specialChars,
					Type: fieldType,
				},
			},
			ColVals: []ColVal{
				generateColVal(t, fieldType),
			},
		}

		var buf []byte
		buf = original.Marshal(buf)

		decoded := &Record{}
		decoded.Unmarshal(buf)

		if decoded.Schema[0].Name != original.Schema[0].Name {
			t.Fatalf("Special character field name mismatch")
		}
	})
}
