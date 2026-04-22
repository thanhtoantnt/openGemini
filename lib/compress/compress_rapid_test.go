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

package compress

import (
	"bytes"
	"math/rand"
	"testing"

	"pgregory.net/rapid"

	"github.com/openGemini/openGemini/lib/record"
	"github.com/openGemini/openGemini/lib/util"
)

func generateFloat64Data(t *rapid.T, minSize, maxSize int) []float64 {
	size := rapid.IntRange(minSize, maxSize).Draw(t, "size")
	values := make([]float64, size)
	for i := range values {
		values[i] = rapid.Float64Range(-1e100, 1e100).Draw(t, "value")
	}
	return values
}

func generateFloat64Array(t *rapid.T) []byte {
	values := generateFloat64Data(t, 1, 1000)
	return util.Float64Slice2byte(values)
}

func TestRLE_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rle := NewRLE(8)
		original := generateFloat64Array(t)

		encoded := make([]byte, 0, len(original))
		encoded, err := rle.Encoding(original, encoded)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		decoded := make([]byte, 0, len(original))
		decoded, err = rle.Decoding(encoded, decoded)
		if err != nil {
			t.Fatalf("Decoding failed: %v", err)
		}

		if !bytes.Equal(original, decoded) {
			t.Fatalf("Roundtrip failed: original len=%d, decoded len=%d", len(original), len(decoded))
		}
	})
}

func TestRLE_SameValueRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rle := NewRLE(8)

		size := rapid.IntRange(1, 1000).Draw(t, "size")
		value := rapid.Float64().Draw(t, "value")

		values := make([]float64, size)
		for i := range values {
			values[i] = value
		}
		original := util.Float64Slice2byte(values)

		encoded := make([]byte, 0, len(original))
		encoded, err := rle.SameValueEncoding(original, encoded)
		if err != nil {
			t.Fatalf("SameValueEncoding failed: %v", err)
		}

		decoded := make([]byte, 0, len(original))
		decoded, err = rle.SameValueDecoding(encoded, decoded)
		if err != nil {
			t.Fatalf("SameValueDecoding failed: %v", err)
		}

		if !bytes.Equal(original, decoded) {
			t.Fatalf("SameValue roundtrip failed for value=%f, size=%d", value, size)
		}
	})
}

func TestSnappy_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := rapid.SliceOf(rapid.Byte()).Draw(t, "data")

		encoded, err := SnappyEncoding(original, nil)
		if err != nil {
			t.Fatalf("SnappyEncoding failed: %v", err)
		}

		decoded, err := SnappyDecoding(encoded, nil)
		if err != nil {
			t.Fatalf("SnappyDecoding failed: %v", err)
		}

		if !bytes.Equal(original, decoded) {
			t.Fatalf("Snappy roundtrip failed: original len=%d, decoded len=%d", len(original), len(decoded))
		}

		if len(encoded) > len(original) {
			t.Logf("Note: Snappy expanded data from %d to %d bytes", len(original), len(encoded))
		}
	})
}

func TestGorilla_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := generateFloat64Array(t)

		encoded, err := GorillaEncoding(original, nil)
		if err != nil {
			t.Fatalf("GorillaEncoding failed: %v", err)
		}

		decoded, err := GorillaDecoding(encoded, nil)
		if err != nil {
			t.Fatalf("GorillaDecoding failed: %v", err)
		}

		if !bytes.Equal(original, decoded) {
			origVals := util.Bytes2Float64Slice(original)
			decVals := util.Bytes2Float64Slice(decoded)
			t.Fatalf("Gorilla roundtrip failed: original=%v, decoded=%v", origVals, decVals)
		}
	})
}

func TestGorilla_MonotonicSeries(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.IntRange(2, 1000).Draw(t, "size")
		start := rapid.Int64Range(0, 1e6).Draw(t, "start")
		delta := rapid.Int64Range(1, 100).Draw(t, "delta")

		values := make([]float64, size)
		for i := range values {
			values[i] = float64(start + int64(i)*delta)
		}
		original := util.Float64Slice2byte(values)

		encoded, err := GorillaEncoding(original, nil)
		if err != nil {
			t.Fatalf("GorillaEncoding failed for monotonic series: %v", err)
		}

		decoded, err := GorillaDecoding(encoded, nil)
		if err != nil {
			t.Fatalf("GorillaDecoding failed for monotonic series: %v", err)
		}

		if !bytes.Equal(original, decoded) {
			t.Fatalf("Gorilla roundtrip failed for monotonic series")
		}

		compressionRatio := float64(len(encoded)) / float64(len(original))
		if compressionRatio > 0.5 {
			t.Logf("Warning: Poor compression ratio for monotonic series: %.2f", compressionRatio)
		}
	})
}

func TestRLE_RandomRuns(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rle := NewRLE(8)

		numRuns := rapid.IntRange(1, 100).Draw(t, "numRuns")
		values := []float64{}

		for i := 0; i < numRuns; i++ {
			runLength := rapid.IntRange(1, 50).Draw(t, "runLength")
			value := rapid.Float64().Draw(t, "value")

			for j := 0; j < runLength; j++ {
				values = append(values, value)
			}
		}

		original := util.Float64Slice2byte(values)

		encoded := make([]byte, 0, len(original))
		encoded, err := rle.Encoding(original, encoded)
		if err != nil {
			t.Fatalf("RLE Encoding failed: %v", err)
		}

		decoded := make([]byte, 0, len(original))
		decoded, err = rle.Decoding(encoded, decoded)
		if err != nil {
			t.Fatalf("RLE Decoding failed: %v", err)
		}

		if !bytes.Equal(original, decoded) {
			t.Fatalf("RLE roundtrip failed for random runs")
		}
	})
}

func TestCompression_Decompression_Invariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		method := rapid.SampledFrom([]string{"snappy", "gorilla"}).Draw(t, "method")
		original := rapid.SliceOfN(rapid.Byte(), 1, 10000).Draw(t, "data")

		var encoded, decoded []byte
		var err error

		switch method {
		case "snappy":
			encoded, err = SnappyEncoding(original, nil)
			if err != nil {
				t.Fatalf("SnappyEncoding failed: %v", err)
			}
			decoded, err = SnappyDecoding(encoded, nil)
			if err != nil {
				t.Fatalf("SnappyDecoding failed: %v", err)
			}
		case "gorilla":
			values := make([]float64, len(original)/8)
			if len(values) > 0 {
				for i := range values {
					values[i] = rand.Float64()
				}
				original = util.Float64Slice2byte(values)
			}
			encoded, err = GorillaEncoding(original, nil)
			if err != nil {
				t.Fatalf("GorillaEncoding failed: %v", err)
			}
			decoded, err = GorillaDecoding(encoded, nil)
			if err != nil {
				t.Fatalf("GorillaDecoding failed: %v", err)
			}
		}

		if !bytes.Equal(original, decoded) {
			t.Fatalf("%s: roundtrip failed", method)
		}
	})
}

func TestRLE_EmptyInput(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rle := NewRLE(8)

		empty := []byte{}

		encoded, err := rle.Encoding(empty, nil)
		if err != nil {
			t.Fatalf("RLE Encoding of empty should not error: %v", err)
		}

		decoded, err := rle.Decoding(encoded, nil)
		if err != nil {
			t.Fatalf("RLE Decoding of empty should not error: %v", err)
		}

		if !bytes.Equal(empty, decoded) {
			t.Fatalf("RLE empty roundtrip failed")
		}
	})
}

func TestCompression_Idempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := generateFloat64Array(t)
		rle := NewRLE(8)

		firstEncoded, err := rle.Encoding(data, nil)
		if err != nil {
			t.Fatalf("First encoding failed: %v", err)
		}

		secondEncoded, err := rle.Encoding(firstEncoded, nil)
		if err != nil {
			t.Fatalf("Second encoding failed: %v", err)
		}

		if bytes.Equal(firstEncoded, secondEncoded) {
			t.Logf("Note: RLE encoding appears idempotent (both outputs equal)")
		}
	})
}
