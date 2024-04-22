package bytecode

import (
	"math"
	"testing"

	"evylang.dev/evy/pkg/assert"
	"evylang.dev/evy/pkg/parser"
)

// testCase covers both the compiler and the VM.
type testCase struct {
	name string
	// input is an evy program
	input string
	// wantStackTop is the expected last popped element of the stack in the vm.
	wantStackTop value
}

func TestVMGlobals(t *testing.T) {
	tests := []testCase{
		{
			name:         "inferred declaration",
			input:        "x := 1",
			wantStackTop: makeValue(t, 1),
		},
		{
			name: "assignment",
			input: `x := 1
			y := x
			y = x + 1
			y = y`,
			wantStackTop: makeValue(t, 2),
		},
		{
			name: "index assignment",
			input: `x := [1 2 3]
			x[0] = x[2]
			x[2] = 1
			x = x`,
			wantStackTop: makeValue(t, []any{3, 2, 1}),
		},
		{
			name: "nested index assignment",
			input: `x := [[1 2] [3 4]]
			x[0][0] = x[0][1]
			x = x`,
			wantStackTop: makeValue(t, []any{[]any{2, 2}, []any{3, 4}}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")

			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func TestUserError(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "divide by zero",
			input:   "x := 2 / 0",
			wantErr: ErrDivideByZero,
		}, {
			name:    "modulo by zero",
			input:   "x := 2 % 0",
			wantErr: ErrDivideByZero,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)

			vm := NewVM(bytecode)
			err := vm.Run()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestBoolExpressions(t *testing.T) {
	tests := []testCase{
		{
			name:         "literal true",
			input:        "x := true",
			wantStackTop: makeValue(t, true),
		},
		{
			name:         "literal false",
			input:        "x := false",
			wantStackTop: makeValue(t, false),
		},
		{
			name:         "not operator",
			input:        "x := !true",
			wantStackTop: makeValue(t, false),
		},
		{
			name:         "equal operator",
			input:        "x := 1 == 1",
			wantStackTop: makeValue(t, true),
		},
		{
			name:         "not operator",
			input:        "x := 1 != 1",
			wantStackTop: makeValue(t, false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")

			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func TestNumOperations(t *testing.T) {
	tests := []testCase{
		{
			name:         "addition",
			input:        "x := 2 + 1",
			wantStackTop: makeValue(t, 3),
		},
		{
			name:         "subtraction",
			input:        "x := 2 - 1",
			wantStackTop: makeValue(t, 1),
		},
		{
			name:         "multiplication",
			input:        "x := 2 * 1",
			wantStackTop: makeValue(t, 2),
		},
		{
			name:         "division",
			input:        "x := 2 / 1",
			wantStackTop: makeValue(t, 2),
		},
		{
			name:         "modulo",
			input:        "x := 2 % 1",
			wantStackTop: makeValue(t, 0),
		},
		{
			name:         "float modulo",
			input:        "x := 2.5 % 1.3",
			wantStackTop: makeValue(t, 1.2),
		},
		{
			name:         "minus operator",
			input:        "x := -1",
			wantStackTop: makeValue(t, -1),
		},
		{
			name:         "all operators",
			input:        "x := 1 + 2 - 3 * 4 / 5 % 6",
			wantStackTop: makeValue(t, 1+2-math.Mod(3.0*4.0/5.0, 6.0)),
		},
		{
			name:         "grouped expressions",
			input:        "x := (1 + 2 - 3) * 4 / 5 % 6",
			wantStackTop: makeValue(t, (1+2-3)*4/math.Mod(5.0, 6.0)),
		},
		{
			name:         "less than",
			input:        "x := 1 < 2",
			wantStackTop: makeValue(t, true),
		},
		{
			name:         "less than or equal",
			input:        "x := 1 <= 2",
			wantStackTop: makeValue(t, true),
		},
		{
			name:         "greater than",
			input:        "x := 1 > 2",
			wantStackTop: makeValue(t, false),
		},
		{
			name:         "greater than or equal",
			input:        "x := 1 >= 2",
			wantStackTop: makeValue(t, false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")
			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func TestStringExpressions(t *testing.T) {
	tests := []testCase{
		{
			name:         "assignment",
			input:        `x := "a"`,
			wantStackTop: makeValue(t, "a"),
		},
		{
			name:         "concatenate",
			input:        `x := "a" + "b"`,
			wantStackTop: makeValue(t, "ab"),
		},
		{
			name:         "less than",
			input:        `x := "a" < "b"`,
			wantStackTop: makeValue(t, true),
		},
		{
			name:         "less than or equal",
			input:        `x := "a" <= "b"`,
			wantStackTop: makeValue(t, true),
		},
		{
			name:         "greater than",
			input:        `x := "a" > "b"`,
			wantStackTop: makeValue(t, false),
		},
		{
			name:         "greater than or equal",
			input:        `x := "a" >= "b"`,
			wantStackTop: makeValue(t, false),
		},
		{
			name:         "index",
			input:        `x := "abc"[0]`,
			wantStackTop: makeValue(t, "a"),
		},
		{
			name:         "negative index",
			input:        `x := "abc"[-1]`,
			wantStackTop: makeValue(t, "c"),
		},
		{
			name:         "slice",
			input:        `x := "abc"[1:3]`,
			wantStackTop: makeValue(t, "bc"),
		},
		{
			name:         "negative slice",
			input:        `x := "abc"[-3:-1]`,
			wantStackTop: makeValue(t, "ab"),
		},
		{
			name:         "slice default start",
			input:        `x := "abc"[:1]`,
			wantStackTop: makeValue(t, "a"),
		},
		{
			name:         "slice default end",
			input:        `x := "abc"[1:]`,
			wantStackTop: makeValue(t, "bc"),
		},
		{
			name:         "slice start equals length",
			input:        `x := "abc"[3:]`,
			wantStackTop: makeValue(t, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")
			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func TestArrays(t *testing.T) {
	tests := []testCase{
		{
			name:         "empty assignment",
			input:        "x := []",
			wantStackTop: makeValue(t, []any{}),
		},
		{
			name:         "assignment",
			input:        "x := [1 2 3]",
			wantStackTop: makeValue(t, []any{1, 2, 3}),
		},
		{
			name:         "concatenate",
			input:        "x := [1 2] + [3 4]",
			wantStackTop: makeValue(t, []any{1, 2, 3, 4}),
		},
		{
			name:         "repetition",
			input:        `x := [1 2] * 3`,
			wantStackTop: makeValue(t, []any{1, 2, 1, 2, 1, 2}),
		},
		{
			name:         "zero repetition",
			input:        `x := [1 2] * 0`,
			wantStackTop: makeValue(t, []any{}),
		},
		{
			name: "concatenate preserve original",
			input: `x := [1 2]
			y := [3 4]
			y = x + y
			x = x`,
			wantStackTop: makeValue(t, []any{1, 2}),
		},
		{
			name:         "index",
			input:        `x := [1 2 3][0]`,
			wantStackTop: makeValue(t, 1),
		},
		{
			name:         "negative index",
			input:        `x := [1 2 3][-1]`,
			wantStackTop: makeValue(t, 3),
		},
		{
			name:         "slice",
			input:        `x := [1 2 3][1:3]`,
			wantStackTop: makeValue(t, []any{2, 3}),
		},
		{
			name:         "negative slice",
			input:        `x := [1 2 3][-3:-1]`,
			wantStackTop: makeValue(t, []any{1, 2}),
		},
		{
			name:         "slice default start",
			input:        `x := [1 2 3][:1]`,
			wantStackTop: makeValue(t, []any{1}),
		},
		{
			name:         "slice default end",
			input:        `x := [1 2 3][1:]`,
			wantStackTop: makeValue(t, []any{2, 3}),
		},
		{
			name:         "slice start equals length",
			input:        `x := [1 2 3][3:]`,
			wantStackTop: makeValue(t, []any{}),
		},
		{
			name: "slice preserve original",
			input: `x := [1 2 3]
			y := x[1:]
			y[0] = 8
			x = x`,
			wantStackTop: makeValue(t, []any{1, 2, 3}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")
			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func TestErrArrayRepetition(t *testing.T) {
	type repeatTest struct {
		name  string
		input string
	}
	tests := []repeatTest{
		{
			name:  "non-integer repetition",
			input: `x := [1 2 3] * 4.5`,
		},
		{
			name:  "negative repetition",
			input: `x := [1 2 3] * -1`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.Error(t, ErrBadRepetition, err)
		})
	}
}

func TestErrBounds(t *testing.T) {
	type boundsTest struct {
		name  string
		input string
	}
	tests := []boundsTest{
		{
			name:  "string index out of bounds",
			input: `x := "abc"[3]`,
		},
		{
			name:  "string negative index out of bounds",
			input: `x := "abc"[-4]`,
		},
		{
			name:  "array index out of bounds",
			input: `x := [1 2 3][3]`,
		},
		{
			name:  "array negative index out of bounds",
			input: `x := [1 2 3][-4]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.Error(t, ErrBounds, err)
		})
	}
}

func TestErrIndex(t *testing.T) {
	type boundsTest struct {
		name  string
		input string
	}
	tests := []boundsTest{
		{
			name:  "string index not an integer",
			input: `x := "abc"[1.1]`,
		},
		{
			name:  "array index not an integer",
			input: `x := [1 2 3][1.1]`,
		},
		{
			name:  "array invalid slice type",
			input: `x := [1 2 3][1.1:2.1]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.Error(t, ErrIndexValue, err)
		})
	}
}

func TestErrSlice(t *testing.T) {
	type boundsTest struct {
		name  string
		input string
	}
	tests := []boundsTest{
		{
			name:  "string invalid slice",
			input: `x := "abc"[2:1]`,
		},
		{
			name:  "array invalid slice",
			input: `x := [1 2 3][2:1]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.Error(t, ErrSlice, err)
		})
	}
}

func TestMap(t *testing.T) {
	tests := []testCase{
		{
			name:         "empty",
			input:        "x := {}",
			wantStackTop: makeValue(t, map[string]any{}),
		},
		{
			name:         "assignment",
			input:        "x := {a: 1 b: 2}",
			wantStackTop: makeValue(t, map[string]any{"a": 1, "b": 2}),
		},
		{
			name:         "index",
			input:        `x := {a: 1 b: 2}["b"]`,
			wantStackTop: makeValue(t, 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")
			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func TestErrMapKey(t *testing.T) {
	input := `x := {a: 1}["b"]`
	bytecode := compileBytecode(t, input)
	vm := NewVM(bytecode)
	err := vm.Run()
	assert.Error(t, ErrMapKey, err)
}

func TestIf(t *testing.T) {
	tests := []testCase{
		{
			name: "single if",
			input: `x := 1
			if x == 1
				x = 2
			end`,
			wantStackTop: makeValue(t, 2),
		},
		{
			name: "if else",
			input: `x := 10
			if x < 5
				x = 20
			else
				x = 5
			end`,
			wantStackTop: makeValue(t, 5),
		},
		{
			name: "else if else",
			input: `x := 10
			if x > 10
				x = 10
			else if x > 5
				x = 5
			else
				x = 1
			end`,
			wantStackTop: makeValue(t, 5),
		},
		{
			name: "many elseif",
			input: `x := 3
				if x == 1
					x = 11
				else if x == 2
					x = 12
				else if x == 3
					x = 13
				else 
					x = 14
				end`,
			wantStackTop: makeValue(t, 13),
		},
		{
			name: "no matching if",
			input: `x := 1
			if false
				x = 2
			else if false
				x = 3
			else if false
				x = 4
			end
			x = x`,
			wantStackTop: makeValue(t, 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")
			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func TestWhile(t *testing.T) {
	tests := []testCase{
		{
			name: "enter loop",
			input: `x := 0
			while x < 5
				x = x + 1
			end
			x = x`,
			wantStackTop: makeValue(t, 5),
		},
		{
			name: "skip loop",
			input: `x := 0
			while x > 5
				x = x + 1
			end
			x = x`,
			wantStackTop: makeValue(t, 0),
		},
		{
			name: "break loop",
			input: `x := 0
			while x < 5
				x = x + 1
				if x == 3 
					break
				end
			end
			x = x`,
			wantStackTop: makeValue(t, 3),
		},
		{
			name: "nested loops and breaks",
			input: `
			x := 0
			while true
				while true
					break
				end
				x = x + 1
				break
			end
			x = x`,
			wantStackTop: makeValue(t, 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")
			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

//nolint:maintidx // This function is not complex by any measure.
func TestStepRange(t *testing.T) {
	tests := []testCase{
		{
			name: "for range default start and step",
			input: `x := 0
			for range 10
				x = x + 1
			end
			x = x`,
			wantStackTop: makeValue(t, 10),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 0, 1, 10, 1, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for range 10 (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					mustMake(t, OpConstant, 2),  // 1
					mustMake(t, OpConstant, 3),  // 10
					mustMake(t, OpStepRange),
					mustMake(t, OpJumpOnFalse, 48),
					// x = x + 1
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 0),
					// (hidden i = i + 1)
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 5),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for range default step",
			input: `x := 0
			for range 2 10
				x = x + 1
			end
			x = x`,
			wantStackTop: makeValue(t, 8),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 2, 1, 10, 1, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for range 10 (hidden i := 2)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					mustMake(t, OpConstant, 2),  // 1
					mustMake(t, OpConstant, 3),  // 10
					mustMake(t, OpStepRange),
					mustMake(t, OpJumpOnFalse, 48),
					// x = x + 1
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 0),
					// (hidden i = i + 1)
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 5),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for range var",
			input: `x := 0
			for i := range 10
				x = i
			end
			x = x`,
			wantStackTop: makeValue(t, 9),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 0, 1, 10, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for range 10 (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					mustMake(t, OpConstant, 2),  // 1
					mustMake(t, OpConstant, 3),  // 10
					mustMake(t, OpStepRange),
					mustMake(t, OpJumpOnFalse, 44),
					// x = i
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpSetGlobal, 0),
					// (hidden i = i + 1)
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for range step",
			input: `x := 0
			for i := range 0 10 4
				x = i
			end
			x = x`,
			wantStackTop: makeValue(t, 8),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 0, 4, 10, 4),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for range 10 (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					mustMake(t, OpConstant, 2),  // 4
					mustMake(t, OpConstant, 3),  // 10
					mustMake(t, OpStepRange),
					mustMake(t, OpJumpOnFalse, 44),
					// x = i
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpSetGlobal, 0),
					// (hidden i = i + 4)
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for range negative step",
			input: `x := 0
			for i := range 10 0 -1
				x = i
			end
			x = x`,
			wantStackTop: makeValue(t, 1),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 10, 1, 0, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for range 10 (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					mustMake(t, OpConstant, 2),  // -1
					mustMake(t, OpMinus),
					mustMake(t, OpConstant, 3), // 10
					mustMake(t, OpStepRange),
					mustMake(t, OpJumpOnFalse, 46),
					// x = i
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpSetGlobal, 0),
					// (hidden i = i - 1)
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpMinus),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for range invalid stop",
			input: `x := 0
			for range -10
				x = x + 1
			end
			x = x`,
			wantStackTop: makeValue(t, 0),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 0, 1, 10, 1, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for range 10 (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					mustMake(t, OpConstant, 2),  // 1
					mustMake(t, OpConstant, 3),  // -10
					mustMake(t, OpMinus),
					mustMake(t, OpStepRange),
					mustMake(t, OpJumpOnFalse, 49),
					// x = x + 1
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 0),
					// (hidden i = i + 1)
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 5),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for break",
			input: `x := 0
			for range 5
				x = x + 1
				if x == 3 
                    break
                end
			end
			x = x`,
			wantStackTop: makeValue(t, 3),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 0, 1, 5, 1, 3, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for range 10 (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					mustMake(t, OpConstant, 2),  // 1
					mustMake(t, OpConstant, 3),  // 5
					mustMake(t, OpStepRange),
					mustMake(t, OpJumpOnFalse, 64),
					// x = x + 1
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 0),
					// if x == 3
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpConstant, 5),
					mustMake(t, OpEqual),
					mustMake(t, OpJumpOnFalse, 51),
					// break
					mustMake(t, OpJump, 64),
					// end
					mustMake(t, OpJump, 51),
					// (hidden i = i + 1)
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 6),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			assertBytecode(t, tt.wantBytecode, bytecode)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")
			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func TestArrayRange(t *testing.T) {
	tests := []testCase{
		{
			name: "for range array",
			input: `x := 0
			for e := range [1 2 3]
				x = e
			end
			x = x`,
			wantStackTop: makeValue(t, 3),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 0, 1, 2, 3, 1, 2, 3, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for e := range [1 2 3]
					// (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					// [1 2 3]
					mustMake(t, OpConstant, 2),
					mustMake(t, OpConstant, 3),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpArray, 3),
					mustMake(t, OpIterRange),
					mustMake(t, OpJumpOnFalse, 69),
					// (hidden e := arr[i])
					mustMake(t, OpConstant, 5),
					mustMake(t, OpConstant, 6),
					mustMake(t, OpConstant, 7),
					mustMake(t, OpArray, 3),
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpIndex),
					mustMake(t, OpSetGlobal, 2),
					// x = e
					mustMake(t, OpGetGlobal, 2),
					mustMake(t, OpSetGlobal, 0),
					// i = i + 1
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 8),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for range array no loopvar",
			input: `x := 0
			for range [1 2 3]
				x = x + 1
			end
			x = x`,
			wantStackTop: makeValue(t, 3),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 0, 1, 2, 3, 1, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for range [1 2 3]
					// (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					// [1 2 3]
					mustMake(t, OpConstant, 2),
					mustMake(t, OpConstant, 3),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpArray, 3),
					mustMake(t, OpIterRange),
					mustMake(t, OpJumpOnFalse, 54),
					// x = x + 1
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpConstant, 5),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 0),
					// i = i + 1
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 6),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for range array variable",
			input: `x := 0
			y := [1 2 3]
			for e := range y
				x = e
			end
			x = x`,
			wantStackTop: makeValue(t, 3),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 1, 2, 3, 0, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// y := [1 2 3]
					mustMake(t, OpConstant, 1),
					mustMake(t, OpConstant, 2),
					mustMake(t, OpConstant, 3),
					mustMake(t, OpArray, 3),
					mustMake(t, OpSetGlobal, 1),
					// for e := range y
					// (hidden i := 0)
					mustMake(t, OpConstant, 4),
					mustMake(t, OpSetGlobal, 2),
					mustMake(t, OpGetGlobal, 2), // i
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpIterRange),
					mustMake(t, OpJumpOnFalse, 66),
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpGetGlobal, 2),
					mustMake(t, OpIndex),
					mustMake(t, OpSetGlobal, 3),
					// x = e
					mustMake(t, OpGetGlobal, 3),
					mustMake(t, OpSetGlobal, 0),
					// i = i + 1
					mustMake(t, OpGetGlobal, 2),
					mustMake(t, OpConstant, 5),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 2),
					// end
					mustMake(t, OpJump, 27),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
		{
			name: "for range array break",
			input: `x := 0
			for x := range [1 2 3]
				if x == 2
					break
				end
			end
			x = x`,
			wantStackTop: makeValue(t, 2),
			wantBytecode: &Bytecode{
				Constants: makeValues(t, 0, 0, 1, 2, 3, 1, 2, 3, 2, 1),
				Instructions: makeInstructions(
					// x := 0
					mustMake(t, OpConstant, 0),
					mustMake(t, OpSetGlobal, 0),
					// for e := range [1 2 3]
					// (hidden i := 0)
					mustMake(t, OpConstant, 1),
					mustMake(t, OpSetGlobal, 1),
					mustMake(t, OpGetGlobal, 1), // i
					// [1 2 3]
					mustMake(t, OpConstant, 2),
					mustMake(t, OpConstant, 3),
					mustMake(t, OpConstant, 4),
					mustMake(t, OpArray, 3),
					mustMake(t, OpIterRange),
					mustMake(t, OpJumpOnFalse, 79),
					// (hidden e := arr[i])
					mustMake(t, OpConstant, 5),
					mustMake(t, OpConstant, 6),
					mustMake(t, OpConstant, 7),
					mustMake(t, OpArray, 3),
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpIndex),
					mustMake(t, OpSetGlobal, 0),
					// 	if x == 2
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpConstant, 8),
					mustMake(t, OpEqual),
					mustMake(t, OpJumpOnFalse, 66),
					// 		break
					mustMake(t, OpJump, 79),
					// 	end
					mustMake(t, OpJump, 66),
					// i = i + 1
					mustMake(t, OpGetGlobal, 1),
					mustMake(t, OpConstant, 9),
					mustMake(t, OpAdd),
					mustMake(t, OpSetGlobal, 1),
					// end
					mustMake(t, OpJump, 12),
					// x = x
					mustMake(t, OpGetGlobal, 0),
					mustMake(t, OpSetGlobal, 0),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytecode := compileBytecode(t, tt.input)
			assertBytecode(t, tt.wantBytecode, bytecode)
			vm := NewVM(bytecode)
			err := vm.Run()
			assert.NoError(t, err, "runtime error")
			got := vm.lastPoppedStackElem()
			assert.Equal(t, tt.wantStackTop, got)
		})
	}
}

func compileBytecode(t *testing.T, input string) *Bytecode {
	t.Helper()
	// add x = x to the input so it parses correctly
	input += "\nx = x"
	program, err := parser.Parse(input, parser.Builtins{})
	assert.NoError(t, err, "unexpected parse error")
	comp := NewCompiler()
	err = comp.Compile(program)
	assert.NoError(t, err, "unexpected compile error")
	bc := comp.Bytecode()
	// remove the final 2 instructions that represent x = x
	bc.Instructions = bc.Instructions[:len(bc.Instructions)-6]
	return bc
}

func makeValue(t *testing.T, a any) value {
	t.Helper()
	switch v := a.(type) {
	case []any:
		return arrayVal{Elements: makeValues(t, v...)}
	case map[string]any:
		m := mapVal{}
		for key, val := range v {
			m[key] = makeValue(t, val)
		}
		return m
	case string:
		return stringVal(v)
	case int:
		return numVal(v)
	case float64:
		return numVal(v)
	case bool:
		return boolVal(v)
	default:
		t.Fatalf("makeValue(%q): invalid type: %T", a, a)
		return nil
	}
}

func makeValues(t *testing.T, in ...any) []value {
	t.Helper()
	out := make([]value, len(in))
	for i, a := range in {
		out[i] = makeValue(t, a)
	}
	return out
}
