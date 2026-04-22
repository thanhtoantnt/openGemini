# Property-Based Testing with Rapid for openGemini

This document describes the property-based testing framework implemented using the `rapid` library for the openGemini time series database.

## Overview

Property-based testing (PBT) complements traditional example-based testing by verifying that system properties hold for a wide range of automatically generated inputs. This approach helps uncover edge cases and bugs that might be missed with traditional testing.

## What is Rapid?

Rapid is a modern Go property-based testing library that provides:

- **Type-safe data generation** using generics
- **Automatic minimization** of failing test cases
- **State machine testing** support
- **No dependencies** outside Go standard library
- **Imperative API** similar to standard Go testing

## Installation

Add rapid to your project dependencies:

```bash
go get pgregory.net/rapid
```

## Running Property-Based Tests

Run all property-based tests:

```bash
# Run all rapid tests
go test -v ./lib/compress/... -run Rapid
go test -v ./lib/record/... -run Rapid
go test -v ./lib/strings/... -run Rapid
go test -v ./lib/bufferpool/... -run Rapid

# Run all tests including rapid tests
go test -v ./lib/compress/...
```

Configure rapid behavior:

```bash
# Increase number of test cases (default: 100)
go test -rapid.checks=1000 ./lib/compress/...

# Set random seed for reproducibility
go test -rapid.seed=12345 ./lib/compress/...

# Enable shrinking (default: true)
go test -rapid.shrink=false ./lib/compress/...
```

## Test Coverage

### 1. Compression/Decompression (`lib/compress/compress_rapid_test.go`)

Tests verify lossless compression properties:

- **Roundtrip Property**: Compress → Decompress → Original
  ```go
  TestRLE_Roundtrip        // RLE encoding/decoding
  TestSnappy_Roundtrip     // Snappy compression
  TestGorilla_Roundtrip    // Gorilla compression
  ```

- **Special Case Properties**:
  - Empty inputs
  - Monotonic series
  - Random runs (for RLE)
  - All duplicates

- **Invariant Properties**:
  - Idempotence
  - Size preservation

### 2. Record Marshaling (`lib/record/record_rapid_test.go`)

Tests verify serialization properties:

- **Roundtrip Property**: Marshal → Unmarshal → Original
  ```go
  TestRecord_MarshalUnmarshal_Roundtrip
  ```

- **Edge Cases**:
  - Empty records
  - Single field records
  - Large records (50-100 fields)
  - Records with nulls
  - Special characters in field names

- **Consistency Properties**:
  - CodecSize accuracy
  - Deterministic marshaling

### 3. String Utilities (`lib/strings/strings_rapid_test.go`)

Tests verify string operation properties:

- **UnionSlice Properties**:
  - No duplicates in result
  - All original elements present
  - Idempotence (UnionSlice(UnionSlice(x)) == UnionSlice(x))
  - Handles empty and single-element inputs
  - Preserves already-unique inputs
  - Handles mixed duplicates correctly

- **SortIsEqual Properties**:
  - Works on sorted arrays
  - Different lengths return false
  - Different content returns false
  - Handles empty arrays

### 4. Time Series Operations (`lib/record/timeseries_rapid_test.go`)

Tests verify time series aggregation properties:

- **Aggregate Properties**:
  - Min ≤ Max (always)
  - Min ≤ Avg ≤ Max (when valid)
  - Count matches non-null elements
  - Range consistency

- **Null Handling**:
  - Null count consistency
  - Null position invariants
  - Proper type-specific operations

- **Mathematical Properties**:
  - Sum commutativity
  - Partition property (sum(parts) == sum(whole))
  - Identity property

### 5. Aggregation Properties (`lib/record/aggregation_rapid_test.go`)

Advanced property tests for aggregations:

- **Functional Properties**:
  - Min ≤ Avg ≤ Max (when count > 0)
  - Min ≤ Max (always)
  - Distribution transformations

- **Algebraic Properties**:
  - Partition property
  - Identity property
  - Zero sum property
  - Linearity property
  - Boundedness property

- **Monotonicity**:
  - Sequence ordering properties

### 6. Buffer Pool State Machine (`lib/bufferpool/bufferpool_rapid_test.go`)

State machine testing for concurrent resource management:

- **Operations**:
  - Get buffers
  - Put buffers
  - Resize buffers
  - Concurrent operations

- **Properties**:
  - Capacity constraints
  - Memory reuse
  - Data preservation on resize
  - Thread safety

## Writing New Property-Based Tests

### Basic Pattern

```go
func TestYourFunction_PropName(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate input using rapid generators
        input := rapid.SomeGenerator().Draw(t, "input")
        
        // Call your function
        output := YourFunction(input)
        
        // Verify property
        if !PropertyHolds(output) {
            t.Fatalf("Property violated: input=%v, output=%v", input, output)
        }
    })
}
```

### Common Generators

```go
// Basic types
rapid.Int()                    // Any integer
rapid.IntRange(min, max)      // Integer in range
rapid.Float64()                // Any float
rapid.Float64Range(min, max)  // Float in range
rapid.String()                 // Any string
rapid.Bool()                   // Boolean
rapid.Byte()                   // Byte

// Collections
rapid.SliceOf(generator)       // Slice of values
rapid.SliceOfN(generator, min, max)  // Slice with size range
rapid.MapOf(keyGen, valGen)    // Map

// Custom
rapid.SampledFrom([]T{...})    // From slice
rapid.StringMatching("regex")  // String matching regex
```

### Common Properties

#### 1. Roundtrip (Encode/Decode)
```go
original := generateInput(t)
encoded := encode(original)
decoded := decode(encoded)

if !deepEqual(original, decoded) {
    t.Fatalf("Roundtrip failed")
}
```

#### 2. Idempotence
```go
firstResult := yourFunction(input)
secondResult := yourFunction(firstResult)

if firstResult != secondResult {
    t.Fatalf("Function should be idempotent")
}
```

#### 3. Commutativity
```go
result1 := yourFunction(a, b)
result2 := yourFunction(b, a)

if result1 != result2 {
    t.Fatalf("Function should be commutative")
}
```

#### 4. Associativity
```go
result1 := yourFunction(a, yourFunction(b, c))
result2 := yourFunction(yourFunction(a, b), c)

if result1 != result2 {
    t.Fatalf("Function should be associative")
}
```

#### 5. Identity
```go
identityValue := getIdentity()
result := yourFunction(identityValue, input)

if result != input {
    t.Fatalf("Identity property violated")
}
```

### State Machine Testing

```go
type MyStateMachine struct {
    model *Model
    impl  *Implementation
}

func (m *MyStateMachine) Init(t *rapid.T) {
    m.model = NewModel()
    m.impl = NewImplementation()
}

func (m *MyStateMachine) Check(t *rapid.T) {
    // Verify model matches implementation
}

func (m *MyStateMachine) Action1(t *rapid.T) {
    input := rapid.SomeGenerator().Draw(t, "input")
    m.model.Action1(input)
    m.impl.Action1(input)
}

func TestMyStateMachine(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        machine := &MyStateMachine{}
        rapid.StateMachineActions(t, machine)
    })
}
```

## Best Practices

### 1. **Start Simple**
Begin with basic properties and gradually add complexity:
- Roundtrip → Idempotence → Mathematical properties

### 2. **Use Meaningful Property Names**
```go
// Good
TestCompression_Roundtrip
TestUnionSlice_NoDuplicates

// Avoid
TestCompress1
TestStrings2
```

### 3. **Handle Edge Cases**
```go
rapid.Check(t, func(t *rapid.T) {
    input := rapid.SliceOfN(rapid.Byte(), 0, 1000).Draw(t, "input")
    // Handles empty arrays automatically
})
```

### 4. **Use Appropriate Generator Ranges**
```go
// Good: Reasonable range for testing
rapid.IntRange(1, 1000)

// Avoid: Too large (slow) or too small (weak tests)
rapid.IntRange(1, 1000000)
rapid.IntRange(1, 2)
```

### 5. **Test Invariants, Not Implementations**
```go
// Good: Test property
if !sorted(result) {
    t.Fatalf("Result should be sorted")
}

// Avoid: Test implementation detail
if len(result) == len(input)/2 {
    t.Fatalf("Assumed specific compression ratio")
}
```

### 6. **Use State Machine Testing for Complex Objects**
State machine testing is ideal for:
- Data structures with complex state
- APIs with multiple operations
- Concurrent systems

## Debugging Failed Tests

When a property-based test fails, rapid automatically minimizes the failing case:

1. **Check the minimized example** - it shows the smallest input that fails
2. **Add targeted unit tests** - create traditional tests for the failing case
3. **Fix the bug** - address the root cause
4. **Verify the fix** - run the property test again

## Integration with CI/CD

Add property-based tests to your CI pipeline:

```yaml
test:
  script:
    - go test -v ./lib/compress/... -run Rapid
    - go test -v ./lib/record/... -run Rapid
    - go test -v ./lib/strings/... -run Rapid
    - go test -v ./lib/bufferpool/... -run Rapid
```

## Performance Considerations

- Property-based tests can be slower than unit tests
- Run them separately from unit tests in CI
- Use `-rapid.checks` to control test count
- Consider running comprehensive property tests nightly

## Extending the Framework

To add new property-based tests:

1. **Identify good candidates**:
   - Functions with clear invariants
   - Data structures
   - Serialization/encoding
   - Mathematical operations

2. **Choose appropriate properties**:
   - Roundtrip (encode/decode)
   - Idempotence
   - Commutativity
   - Associativity
   - Identity
   - Monotonicity
   - Boundedness

3. **Write the test**:
   ```bash
   touch lib/yourmodule/yourmodule_rapid_test.go
   ```

4. **Run and verify**:
   ```bash
   go test -v ./lib/yourmodule/... -run Rapid
   ```

## Resources

- [Rapid Library](https://github.com/flyingmutant/rapid)
- [Property-Based Testing in Go](https://www.corylanou.com/blog/2022/05/03/property-based-testing-in-go/)
- [Hypothesis Python Library](https://hypothesis.works/) (inspiration for Rapid)

## Contributing

When adding new property-based tests:

1. Follow the existing naming convention: `*rapid_test.go`
2. Include clear property descriptions in test names
3. Use appropriate generators for your data types
4. Test edge cases explicitly
5. Add documentation for complex properties

## Summary

Property-based testing with Rapid provides a powerful way to:

- **Discover bugs** in edge cases
- **Verify invariants** automatically
- **Improve confidence** in code correctness
- **Complement traditional testing** approaches

The tests in this framework cover core openGemini functionality including compression, serialization, string operations, time series aggregations, and concurrent resource management.