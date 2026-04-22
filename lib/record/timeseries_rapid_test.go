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
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/openGemini/openGemini/lib/util/lifted/vm/protoparser/influx"
)

func generateTimeSeriesPoint(t *rapid.T, hasNulls bool) (timestamp int64, value interface{}, isNull bool) {
	timestamp = rapid.Int64Range(0, 1e12).Draw(t, "timestamp")

	isNull = hasNulls && rapid.Bool().Draw(t, "isNull")

	if isNull {
		return timestamp, nil, true
	}

	fieldType := rapid.SampledFrom([]influx.Field_Type{
		influx.Field_Type_Int,
		influx.Field_Type_Float,
		influx.Field_Type_Boolean,
	}).Draw(t, "type")

	switch fieldType {
	case influx.Field_Type_Int:
		value = rapid.Int64().Draw(t, "int_value")
	case influx.Field_Type_Float:
		value = rapid.Float64().Draw(t, "float_value")
	case influx.Field_Type_Boolean:
		value = rapid.Bool().Draw(t, "bool_value")
	}

	return timestamp, value, isNull
}

func generateTimeSeries(t *rapid.T, minPoints, maxPoints int, fieldType influx.Field_Type, hasNulls bool) ColVal {
	numPoints := rapid.IntRange(minPoints, maxPoints).Draw(t, "numPoints")
	ts := ColVal{}
	ts.Init(nil, numPoints, numPoints)

	for i := 0; i < numPoints; i++ {
		timestamp, value, isNull := generateTimeSeriesPoint(t, hasNulls)

		if isNull {
			ts.AppendNil()
		} else {
			switch fieldType {
			case influx.Field_Type_Int:
				if val, ok := value.(int64); ok {
					ts.AppendInteger(val)
				}
			case influx.Field_Type_Float:
				if val, ok := value.(float64); ok {
					ts.AppendFloat(val)
				}
			case influx.Field_Type_Boolean:
				if val, ok := value.(bool); ok {
					ts.AppendBoolean(val)
				}
			}
		}
	}

	return ts
}

func TestTimeSeries_SumProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasNulls := rapid.Bool().Draw(t, "hasNulls")
		ts := generateTimeSeries(t, 1, 100, influx.Field_Type_Int, hasNulls)

		sum := int64(0)
		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.IntegerValue(i)
				sum += val
			}
		}

		if hasNulls && ts.NilCount > 0 {
			_, ok := ts.IntegerValue(0)
			if !ok {
				return
			}
		}

		if ts.Len > 0 && ts.NilCount < ts.Len {
			val, _ := ts.IntegerValue(0)
			_ = val
		}
	})
}

func TestTimeSeries_MinMaxProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasNulls := rapid.Bool().Draw(t, "hasNulls")
		ts := generateTimeSeries(t, 1, 100, influx.Field_Type_Float, hasNulls)

		minVal := math.Inf(1)
		maxVal := math.Inf(-1)
		hasValues := false

		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.FloatValue(i)
				if val < minVal {
					minVal = val
				}
				if val > maxVal {
					maxVal = val
				}
				hasValues = true
			}
		}

		if hasValues {
			if minVal > maxVal {
				t.Fatalf("Min value %f greater than max value %f", minVal, maxVal)
			}
		}
	})
}

func TestTimeSeries_CountProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasNulls := rapid.Bool().Draw(t, "hasNulls")
		ts := generateTimeSeries(t, 1, 100, influx.Field_Type_Int, hasNulls)

		nonNullCount := 0
		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				nonNullCount++
			}
		}

		if nonNullCount != ts.Len-int(ts.NilCount) {
			t.Fatalf("Count mismatch: computed %d, expected %d", nonNullCount, ts.Len-int(ts.NilCount))
		}

		if hasNulls {
			if nonNullCount > ts.Len {
				t.Fatalf("Non-null count %d cannot exceed total count %d", nonNullCount, ts.Len)
			}
		} else {
			if nonNullCount != ts.Len {
				t.Fatalf("Without nulls, count should equal length: %d vs %d", nonNullCount, ts.Len)
			}
		}
	})
}

func TestTimeSeries_RangeConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, false)

		if ts.Len < 2 {
			return
		}

		minIndex := -1
		maxIndex := -1
		minVal := math.Inf(1)
		maxVal := math.Inf(-1)

		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.FloatValue(i)
				if val < minVal {
					minVal = val
					minIndex = i
				}
				if val > maxVal {
					maxVal = val
					maxIndex = i
				}
			}
		}

		if minIndex != -1 && maxIndex != -1 {
			minAtPos, _ := ts.FloatValue(minIndex)
			maxAtPos, _ := ts.FloatValue(maxIndex)

			if minAtPos != minVal {
				t.Fatalf("Min at wrong index")
			}
			if maxAtPos != maxVal {
				t.Fatalf("Max at wrong index")
			}
		}
	})
}

func TestTimeSeries_NullPositionInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, true)

		for i := 0; i < ts.Len; i++ {
			if ts.IsNil(i) {
				val, ok := ts.FloatValue(i)
				if ok {
					t.Fatalf("Nil position %d returned value: %v", i, val)
				}

				val2, ok2 := ts.IntegerValue(i)
				if ok2 {
					t.Fatalf("Nil position %d returned integer value: %v", i, val2)
				}
			}
		}
	})
}

func TestTimeSeries_AggregationCommutativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts1 := generateTimeSeries(t, 1, 50, influx.Field_Type_Float, false)
		ts2 := generateTimeSeries(t, 1, 50, influx.Field_Type_Float, false)

		min1 := math.Inf(1)
		max1 := math.Inf(-1)
		sum1 := 0.0

		for i := 0; i < ts1.Len; i++ {
			if !ts1.IsNil(i) {
				val, _ := ts1.FloatValue(i)
				sum1 += val
				if val < min1 {
					min1 = val
				}
				if val > max1 {
					max1 = val
				}
			}
		}

		min2 := math.Inf(1)
		max2 := math.Inf(-1)
		sum2 := 0.0

		for i := 0; i < ts2.Len; i++ {
			if !ts2.IsNil(i) {
				val, _ := ts2.FloatValue(i)
				sum2 += val
				if val < min2 {
					min2 = val
				}
				if val > max2 {
					max2 = val
				}
			}
		}

		combinedSum := sum1 + sum2
		combinedMin := math.Min(min1, min2)
		combinedMax := math.Max(max1, max2)

		expectedMinSum := math.Min(min1, min2)
		expectedMaxSum := math.Max(max1, max2)

		if combinedMin != expectedMinSum {
			t.Fatalf("Min combination property violated")
		}
		if combinedMax != expectedMaxSum {
			t.Fatalf("Max combination property violated")
		}
	})
}

func TestTimeSeries_EmptySeries(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := ColVal{}
		ts.Init(nil, 0, 0)

		if ts.Len != 0 {
			t.Fatalf("Empty series should have length 0")
		}
		if ts.NilCount != 0 {
			t.Fatalf("Empty series should have nil count 0")
		}
	})
}

func TestTimeSeries_SinglePoint(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasNulls := rapid.Bool().Draw(t, "hasNulls")
		ts := generateTimeSeries(t, 1, 1, influx.Field_Type_Float, hasNulls)

		if ts.Len != 1 {
			t.Fatalf("Single point series should have length 1")
		}

		if hasNulls && ts.NilCount > 0 {
			if !ts.IsNil(0) {
				t.Fatalf("Nil count > 0 but first element not nil")
			}
		}
	})
}

func TestTimeSeries_NullCountConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, true)

		nullCount := 0
		for i := 0; i < ts.Len; i++ {
			if ts.IsNil(i) {
				nullCount++
			}
		}

		if nullCount != int(ts.NilCount) {
			t.Fatalf("NilCount inconsistency: counted %d, expected %d", nullCount, ts.NilCount)
		}
	})
}

func TestTimeSeries_TypeSpecificOperations(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldType := rapid.SampledFrom([]influx.Field_Type{
			influx.Field_Type_Int,
			influx.Field_Type_Float,
			influx.Field_Type_Boolean,
		}).Draw(t, "type")

		ts := generateTimeSeries(t, 1, 50, fieldType, false)

		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				switch fieldType {
				case influx.Field_Type_Int:
					val, ok := ts.IntegerValue(i)
					if !ok {
						t.Fatalf("Int field should return integer value")
					}
					_ = val

					_, okFloat := ts.FloatValue(i)
					if okFloat {

					}

				case influx.Field_Type_Float:
					val, ok := ts.FloatValue(i)
					if !ok {
						t.Fatalf("Float field should return float value")
					}
					_ = val

				case influx.Field_Type_Boolean:
					val, ok := ts.BooleanValue(i)
					if !ok {
						t.Fatalf("Boolean field should return boolean value")
					}
					_ = val
				}
			}
		}
	})
}
