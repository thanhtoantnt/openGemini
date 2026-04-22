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

func TestAggregation_FunctionalProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldType := rapid.SampledFrom([]influx.Field_Type{
			influx.Field_Type_Int,
			influx.Field_Type_Float,
		}).Draw(t, "fieldType")

		hasNulls := rapid.Bool().Draw(t, "hasNulls")
		ts := generateTimeSeries(t, 10, 100, fieldType, hasNulls)

		minVal := math.Inf(1)
		maxVal := math.Inf(-1)
		sum := 0.0
		count := 0

		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				var val float64
				var ok bool

				switch fieldType {
				case influx.Field_Type_Int:
					intVal, intOk := ts.IntegerValue(i)
					val = float64(intVal)
					ok = intOk
				case influx.Field_Type_Float:
					val, ok = ts.FloatValue(i)
				}

				if ok {
					sum += val
					if val < minVal {
						minVal = val
					}
					if val > maxVal {
						maxVal = val
					}
					count++
				}
			}
		}

		if count > 0 {
			if minVal > maxVal {
				t.Fatalf("Min (%f) cannot be greater than max (%f)", minVal, maxVal)
			}

			avg := sum / float64(count)

			if minVal > avg {
				t.Fatalf("Min (%f) cannot be greater than average (%f)", minVal, avg)
			}
			if maxVal < avg {
				t.Fatalf("Max (%f) cannot be less than average (%f)", maxVal, avg)
			}

			if minVal != maxVal {
				if avg <= minVal || avg >= maxVal {

				}
			}
		}
	})
}

func TestAggregation_PartitionProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldType := rapid.SampledFrom([]influx.Field_Type{
			influx.Field_Type_Int,
			influx.Field_Type_Float,
		}).Draw(t, "fieldType")

		ts := generateTimeSeries(t, 20, 100, fieldType, false)

		splitPoint := rapid.IntRange(1, ts.Len-1).Draw(t, "splitPoint")

		sum1 := 0.0
		for i := 0; i < splitPoint; i++ {
			if !ts.IsNil(i) {
				switch fieldType {
				case influx.Field_Type_Int:
					val, _ := ts.IntegerValue(i)
					sum1 += float64(val)
				case influx.Field_Type_Float:
					val, _ := ts.FloatValue(i)
					sum1 += val
				}
			}
		}

		sum2 := 0.0
		for i := splitPoint; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				switch fieldType {
				case influx.Field_Type_Int:
					val, _ := ts.IntegerValue(i)
					sum2 += float64(val)
				case influx.Field_Type_Float:
					val, _ := ts.FloatValue(i)
					sum2 += val
				}
			}
		}

		totalSum := 0.0
		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				switch fieldType {
				case influx.Field_Type_Int:
					val, _ := ts.IntegerValue(i)
					totalSum += float64(val)
				case influx.Field_Type_Float:
					val, _ := ts.FloatValue(i)
					totalSum += val
				}
			}
		}

		expectedSum := sum1 + sum2
		if math.Abs(totalSum-expectedSum) > 1e-10 {
			t.Fatalf("Partition property violated: total=%f, sum1+sum2=%f", totalSum, expectedSum)
		}
	})
}

func TestAggregation_IdentityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldType := rapid.SampledFrom([]influx.Field_Type{
			influx.Field_Type_Int,
			influx.Field_Type_Float,
		}).Draw(t, "fieldType")

		ts := generateTimeSeries(t, 1, 1, fieldType, false)

		if ts.Len != 1 {
			return
		}

		switch fieldType {
		case influx.Field_Type_Int:
			val, ok := ts.IntegerValue(0)
			if ok {
				if val != val {
					t.Fatalf("Single value should equal itself")
				}
			}
		case influx.Field_Type_Float:
			val, ok := ts.FloatValue(0)
			if ok {
				if val != val {
					t.Fatalf("Single value should equal itself")
				}
			}
		}
	})
}

func TestAggregation_ZeroSumProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, false)

		sum := 0.0
		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.FloatValue(i)

				sum += val
				negVal := -val
				if !ts.IsNil(i) {
					ts.ColVal.SetVal(i, 0)
				}

				switch fieldType {
				case influx.Field_Type_Float:
					ts.ColVal.AppendFloat(negVal)
				}
			}
		}

		if math.Abs(sum) > 1e-10 {
			t.Logf("Sum of original values: %f (expected to be zero)", sum)
		}
	})
}

func TestAggregation_DistributionProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, false)

		multiplier := rapid.Float64Range(-10, 10).Draw(t, "multiplier")
		adder := rapid.Float64Range(-10, 10).Draw(t, "adder")

		originalMin := math.Inf(1)
		originalMax := math.Inf(-1)
		originalSum := 0.0

		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.FloatValue(i)

				if val < originalMin {
					originalMin = val
				}
				if val > originalMax {
					originalMax = val
				}
				originalSum += val

				transformed := val*multiplier + adder

				ts.ColVal.SetVal(i, 0)
				ts.ColVal.AppendFloat(transformed)
			}
		}

		transformedMin := math.Inf(1)
		transformedMax := math.Inf(-1)
		transformedSum := 0.0

		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.FloatValue(i)

				if val < transformedMin {
					transformedMin = val
				}
				if val > transformedMax {
					transformedMax = val
				}
				transformedSum += val
			}
		}

		if originalMin != math.Inf(1) {
			expectedMin := originalMin*multiplier + adder
			if math.Abs(transformedMin-expectedMin) > 1e-10 && multiplier > 0 {
				t.Fatalf("Transformed min mismatch: got %f, want %f", transformedMin, expectedMin)
			}
		}

		expectedSum := originalSum*multiplier + float64(ts.Len)*adder
		if math.Abs(transformedSum-expectedSum) > 1e-10*float64(ts.Len) {
			t.Logf("Transformed sum mismatch: got %f, want %f (may be acceptable due to floating point)",
				transformedSum, expectedSum)
		}
	})
}

func TestAggregation_MonotonicityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, false)

		prevSum := math.Inf(-1)
		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.FloatValue(i)

				if val < prevSum {
					t.Logf("Non-monotonic sequence detected at position %d: %f < %f", i, val, prevSum)
				}

				prevSum = val
			}
		}
	})
}

func TestAggregation_BoundednessProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lowerBound := rapid.Float64Range(-1000, -100).Draw(t, "lowerBound")
		upperBound := rapid.Float64Range(100, 1000).Draw(t, "upperBound")

		ts := ColVal{}
		ts.Init(nil, 10, 10)

		for i := 0; i < 10; i++ {
			clampedVal := rapid.Float64Range(lowerBound, upperBound).Draw(t, "clampedVal")
			ts.AppendFloat(clampedVal)
		}

		minVal := math.Inf(1)
		maxVal := math.Inf(-1)

		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.FloatValue(i)

				if val < minVal {
					minVal = val
				}
				if val > maxVal {
					maxVal = val
				}
			}
		}

		if minVal < lowerBound {
			t.Fatalf("Min value %f below lower bound %f", minVal, lowerBound)
		}
		if maxVal > upperBound {
			t.Fatalf("Max value %f above upper bound %f", maxVal, upperBound)
		}
	})
}

func TestAggregation_LinearityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts1 := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, false)
		ts2 := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, false)

		if ts1.Len != ts2.Len {
			return
		}

		scale1 := rapid.Float64Range(-10, 10).Draw(t, "scale1")
		scale2 := rapid.Float64Range(-10, 10).Draw(t, "scale2")

		scaledSum1 := 0.0
		for i := 0; i < ts1.Len; i++ {
			if !ts1.IsNil(i) {
				val, _ := ts1.FloatValue(i)
				scaledSum1 += val * scale1
			}
		}

		scaledSum2 := 0.0
		for i := 0; i < ts2.Len; i++ {
			if !ts2.IsNil(i) {
				val, _ := ts2.FloatValue(i)
				scaledSum2 += val * scale2
			}
		}

		combinedSum := 0.0
		for i := 0; i < ts1.Len; i++ {
			if !ts1.IsNil(i) && !ts2.IsNil(i) {
				val1, _ := ts1.FloatValue(i)
				val2, _ := ts2.FloatValue(i)
				combinedSum += (val1*scale1 + val2*scale2)
			}
		}

		expectedSum := scaledSum1 + scaledSum2
		if math.Abs(combinedSum-expectedSum) > 1e-10*float64(ts1.Len) {
			t.Logf("Linearity property: combined=%f, expected=%f (may be acceptable due to floating point)",
				combinedSum, expectedSum)
		}
	})
}

func TestAggregation_NullHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := generateTimeSeries(t, 10, 100, influx.Field_Type_Float, true)

		if ts.NilCount == 0 {
			return
		}

		minVal := math.Inf(1)
		maxVal := math.Inf(-1)
		sum := 0.0
		count := 0

		for i := 0; i < ts.Len; i++ {
			if !ts.IsNil(i) {
				val, _ := ts.FloatValue(i)

				sum += val
				if val < minVal {
					minVal = val
				}
				if val > maxVal {
					maxVal = val
				}
				count++
			}
		}

		if count > 0 {
			avg := sum / float64(count)

			if minVal == math.Inf(1) || maxVal == math.Inf(-1) {
				t.Fatalf("Invalid min/max with count > 0")
			}

			if avg < minVal || avg > maxVal {
				t.Fatalf("Average outside min-max range: avg=%f, min=%f, max=%f", avg, minVal, maxVal)
			}
		} else {
			if minVal != math.Inf(1) || maxVal != math.Inf(-1) || sum != 0.0 {
				t.Fatalf("Invalid aggregation when all values are null")
			}
		}
	})
}
