package parser

import (
	"errors"
	"sort"
	"strings"
	"testing"

	"evylang.dev/evy/pkg/assert"
)

func TestParseDecl(t *testing.T) {
	tests := map[string][]string{
		"a := 1":     {"a=1"},
		"a:bool":     {"a=false"},
		"\na:bool\n": {"\na=false\n"},
		`a := "abc"
		b:bool
		c := true
		print a b c`: {`a="abc"`, "b=false", "c=true", "print(any(a), any(b), any(c))"},
		"a:[]num":                            {"a=[]"},
		"a:{}[]num":                          {"a={}"},
		"a:{}[]any":                          {"a={}"},
		"a := [true]":                        {"a=[true]"},
		"a := []":                            {"a=[]"},
		"a := [[1 2] ([3 4])]":               {"a=[[1, 2], [3, 4]]"},
		"a := {a:1 b:2}":                     {"a={a:1, b:2}"},
		"a := {digits: [1 2 3] nums: [4 5]}": {"a={digits:[1, 2, 3], nums:[4, 5]}"},
		"a := {digits: [] nums: [4]}":        {"a={digits:[], nums:[4]}"},
		"a := {digits: [4] nums: []}":        {"a={digits:[4], nums:[]}"},
		"a := [{}]":                          {"a=[{}]"},
		"a := {a:1 b:true}":                  {"a={a:any(1), b:any(true)}"},
		"a := {a:1 b:true c:[1]}":            {"a={a:any(1), b:any(true), c:any([1])}"},
		"a := [{a:1}]":                       {"a=[{a:1}]"},
	}
	for input, wantSlice := range tests {
		input += "\n print a"
		wantSlice = append(wantSlice, "print(any(a))")
		want := strings.Join(wantSlice, "\n") + "\n"
		parser := newParser(input, testBuiltins())
		got := parser.parse()
		assertNoParseError(t, parser, input)
		assert.Equal(t, want, got.String())
	}
}

func TestEmptyProgram(t *testing.T) {
	tests := map[string]string{
		"":                "\n",
		"\n":              "\n",
		"\n\n\n":          "\n\n\n",
		" ":               "\n",
		" \n //adf \n":    "\n\n",
		"//blabla":        "\n",
		"//blabla\n":      "\n",
		" \n //blabla \n": "\n\n",
		" \n //blabla":    "\n\n",
	}
	for input, want := range tests {
		parser := newParser(input, testBuiltins())
		got := parser.parse()
		assertNoParseError(t, parser, input)
		assert.Equal(t, want, got.String(), input)
	}
}

func TestParseDeclError(t *testing.T) {
	tests := map[string]string{
		"a :invalid":    `line 1 column 1: invalid type declaration for "a"`,
		"a :":           `line 1 column 1: invalid type declaration for "a"`,
		"a :\n":         `line 1 column 1: invalid type declaration for "a"`,
		"a ://blabla\n": `line 1 column 1: invalid type declaration for "a"`,
		"a :true":       `line 1 column 1: invalid type declaration for "a"`,
		"a :[]":         `line 1 column 1: invalid type declaration for "a"`,
		"a :num[]":      `line 1 column 7: expected end of line, found "["`,
		"a :()":         `line 1 column 1: invalid type declaration for "a"`,
		"a ::":          `line 1 column 1: invalid type declaration for "a"`,
		"a := {}{":      `line 1 column 8: expected end of line, found "{"`,
		"a :=:":         `line 1 column 5: unexpected ":"`,
		"a := {":        `line 1 column 7: expected "}", got end of input`,
		"a := {}[":      `line 1 column 9: unexpected end of input`,
		"a :num num":    `line 1 column 8: expected end of line, found "num"`,
		"a :num{}num":   `line 1 column 7: expected end of line, found "{"`,
		"_ :num":        `line 1 column 1: declaration of anonymous variable "_" not allowed here`,
		"_ := 0":        `line 1 column 1: declaration of anonymous variable "_" not allowed here`,
		`
m := {name: "Greta"}
s := name
print m[s]`: `line 3 column 6: unknown variable name "name"`,
	}
	for input, err1 := range tests {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		got := parser.errors.Truncate(1)
		assert.Equal(t, err1, got.Error(), "input: %s\nerrors:\n%s", input, parser.errors)
	}
}

func TestFunccall(t *testing.T) {
	tests := map[string][]string{
		"print":                          {"print()"},
		"print 123":                      {"print(any(123))"},
		`print 123 "abc"`:                {`print(any(123), any("abc"))`},
		"a:=1 \n print a":                {"a=1", "print(any(a))"},
		`a := len "abc"` + " \n print a": {`a=len(any("abc"))`, "print(any(a))"},
		`len "abc"`:                      {`len(any("abc"))`},
		`len []`:                         {"len(any([]))"},
		"a:string \n print a":            {`a=""`, "print(any(a))"},
		`a:=true
		b:string
		print a b`: {`a=true`, `b=""`, `print(any(a), any(b))`},
	}
	for input, wantSlice := range tests {
		want := strings.Join(wantSlice, "\n") + "\n"
		parser := newParser(input, testBuiltins())
		got := parser.parse()
		assertNoParseError(t, parser, input)
		assert.Equal(t, want, got.String())
	}
}

func TestFunccallError(t *testing.T) {
	builtins := testBuiltins()
	builtins.Funcs["f0"] = &FuncDefStmt{Name: "f0", ReturnType: NONE_TYPE}
	builtins.Funcs["f1"] = &FuncDefStmt{Name: "f1", VariadicParam: &Var{Name: "a", T: NUM_TYPE}, ReturnType: NONE_TYPE}
	builtins.Funcs["f2"] = &FuncDefStmt{Name: "f2", Params: []*Var{{Name: "a", T: NUM_TYPE}}, ReturnType: NONE_TYPE}
	builtins.Funcs["f3"] = &FuncDefStmt{
		Name:       "f3",
		Params:     []*Var{{Name: "a", T: NUM_TYPE}, {Name: "b", T: STRING_TYPE}},
		ReturnType: NONE_TYPE,
	}
	tests := map[string]string{
		`len 2 2`:    `line 1 column 7: "len" takes 1 argument, found 2`,
		`len`:        `line 1 column 4: "len" takes 1 argument, found 0`,
		`a := print`: `line 1 column 11: invalid declaration, function "print" has no return value`,
		`a := f0`:    `line 1 column 8: invalid declaration, function "f0" has no return value`,
		`f0 "arg"`:   `line 1 column 4: "f0" takes 0 arguments, found 1`,
		`f2`:         `line 1 column 3: "f2" takes 1 argument, found 0`,
		`f2 f1`:      `line 1 column 4: function call must be parenthesized: (f1 ...)`,
		`f1 "arg"`:   `line 1 column 4: "f1" takes variadic arguments of type num, found string`,
		`f3 1 2`:     `line 1 column 6: "f3" takes 2nd argument of type string, found num`,
		`f3 "1" "2"`: `line 1 column 4: "f3" takes 1st argument of type num, found string`,
		`foo 0`:      `line 1 column 1: unknown function "foo"`,
	}
	for input, err1 := range tests {
		parser := newParser(input, builtins)
		_ = parser.parse()
		assertParseError(t, parser, input)
		got := parser.errors.Truncate(1)
		assert.Equal(t, err1, got.Error(), "input: %s\nerrors:\n%s", input, parser.errors)
	}
}

func TestBlock(t *testing.T) {
	tests := map[string]string{
		`
if true
	print "TRUE"
end`: `
if (true) {
print(any("TRUE"))
}
`,
		`
if true
	if true
		print "TRUE"
	end
end`: `
if (true) {
if (true) {
print(any("TRUE"))
}
}
`,
	}
	for input, want := range tests {
		parser := newParser(input, testBuiltins())
		got := parser.parse()
		assertNoParseError(t, parser, input)
		assert.Equal(t, want, got.String())
	}
}

func TestToplevelExprFuncCall(t *testing.T) {
	input := `
x := len "123"
print x
`
	parser := newParser(input, testBuiltins())
	got := parser.parse()
	assertNoParseError(t, parser, input)
	want := `
x=len(any("123"))
print(any(x))
`
	assert.Equal(t, want, got.String())
}

func TestFuncDef(t *testing.T) {
	input := `
c := 1
func nums1:num n1:num n2:num
	if c > 10
	    print c
	    return n1
	end
	return n2
end
on down
	if c > 10
	    print c
	end
end
func nums2:num n1:num n2:num
	if c > 10
		return n1
	else
		return n2
	end
end
func nums3
	if true
		return
	end
end
print "success"
func nums4:num
	a := 5
	while true
		return 1
	end
	print a "reachable"
	return 0
end
func nums5 _:num
	print "nums5 not yet implemented"
end
`
	parser := newParser(input, testBuiltins())
	_ = parser.parse()
	assertNoParseError(t, parser, input)
	builtinCnt := len(testBuiltins().Funcs)
	assert.Equal(t, builtinCnt+5, len(parser.funcs))
	got := parser.funcs["nums1"]
	assert.Equal(t, "nums1", got.Name)
	assert.Equal(t, NUM_TYPE, got.ReturnType)
	var wantVariadicParam *Var
	assert.Equal(t, wantVariadicParam, got.VariadicParam)
	assert.Equal(t, 2, len(got.Params))
	n1 := got.Params[0]
	assert.Equal(t, "n1", n1.Name)
	assert.Equal(t, NUM_TYPE, n1.Type())
	assert.Equal(t, 2, len(got.Body.Statements))
	returnStmt := got.Body.Statements[1]
	assert.Equal(t, "return n2", returnStmt.String())
}

func TestVariadicFuncDef(t *testing.T) {
	inputs := []string{
		`
func fox nums:num...
  test nums
end

func test nums:[]num
  print nums
end

fox 1 2 3`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestVariadicFuncDefErr(t *testing.T) {
	inputs := map[string]string{
		`
func fox n:num nums:num...
  print n nums
end

fox 1 2 3`: "line 2 column 27: variadic parameter cannot be used with other parameters",
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestReturn(t *testing.T) {
	inputs := []string{
		`
func fn
    print 1
    return
    // unreachable code!?
end
`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestReturnErr(t *testing.T) {
	inputs := map[string]string{
		`
func add:num
	return 1
	print "boom"
end
`: "line 4 column 2: unreachable code",
		`
func nums:num
	if true
		return 1
	else
		return 2
	end
	print "boom"
end
`: "line 8 column 2: unreachable code",
		`
func nums:num
	if true
		if true
			return 3
		else
			return 4
		end
	else
		return 2
	end
	print "boom"
end
`: "line 12 column 2: unreachable code",
		`
func foo
	while true
		if true
			return
		else
			return
		end
		print "deadcode"
	end
end
`: "line 9 column 3: unreachable code",
		`
foo
return false
func foo
  print "hello"
end
print "do i run?"
`: "line 3 column 8: return statement not allowed here",
		`
func nums:num
	while true
		if true
			return 1
		end
	end
end
`: "line 8 column 1: missing return",
		`
func nums:num
	if true
		return 1
	end
end
`: "line 6 column 1: missing return",
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestAssignment(t *testing.T) {
	inputs := []string{
		`
a := 1
b:num
b = a
print b
`, `
a:num
b:num
b = a
print b
`, `
a:num
b:any
b = a
print b
`, `
a := [0 2 3]
a[0] = 1
print a
`, `
a :=  [ [0 2 3] ([4 5]) ]
a[0][1] = 1
print a
`, `
a := {name: "mali"}
a.sport = "climbing"
print a
`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestAssignmentErr(t *testing.T) {
	inputs := map[string]string{
		`
b:num
b = true
`: `line 3 column 1: "b" accepts values of type num, found bool`,
		`
a:= 1
a = b
`: `line 3 column 5: unknown variable name "b"`,
		`
a:= 1
b = a
`: `line 3 column 1: unknown variable name "b"`,
		`
a:= 1
a = []
`: `line 3 column 1: "a" accepts values of type num, found []`,
		`
a:num
b:any
a = b
`: `line 4 column 1: "a" accepts values of type num, found any`,
		`
m := [{a:1} {b:2}]
m[0]. a = 3
print m`: `line 3 column 5: unexpected whitespace after "."`,
		`
func fn:bool
	return true
end
fn = 3
`: `line 5 column 1: cannot assign to "fn" as it is a function not a variable`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestScope(t *testing.T) {
	inputs := []string{
		`
x := 1
func foo
	x := "abc"
	print x
end
print x
`, `
x := 1
func foo x:string
	x = "abc"
	print x
end
print x
`, `
x := 1
func foo
	x = 2
	print x
end
`, `
x := 1
func foo x:string...
	print x
end
print x
`, `
x := 1
if true
	x := "abc" // block scope
	print x
end
print x
`, `
a := [ ([1 2 3]) ([4 5 6]) ]
b := a[0]
b[1] = 7
print a
`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestUnusedErr(t *testing.T) {
	inputs := map[string]string{
		`
x := 1
`: `line 2 column 1: "x" declared but not used`,
		`
x := 1
if true
	x := 1
end
print x
`: `line 4 column 2: "x" declared but not used`,
		`
x := 1
if true
	x := 1
	print x
end
`: `line 2 column 1: "x" declared but not used`,
		`
x := 1
if true
	print "foo"
else
	x := 1
	print x
end
`: `line 2 column 1: "x" declared but not used`,
		`
x := 1
if true
	print "foo"
else
	x := 1
end
print x
`: `line 6 column 2: "x" declared but not used`,
		`
x := 1
if true
	print "foo"
else if true
	x := 1
end
print x
`: `line 6 column 2: "x" declared but not used`,
		`
x := 1
for i := range 10
	x := 2
	print i x
end
`: `line 2 column 1: "x" declared but not used`,
		`
x := 1
for i := range 10
	x := 2 * i
end
print x
`: `line 4 column 2: "x" declared but not used`,
		`
x := 1
while true
	x := 2
	print x
end
`: `line 2 column 1: "x" declared but not used`,
		`
x := 1
while true
	x := 2
end
print x
`: `line 4 column 2: "x" declared but not used`,
		`
x := 1
func foo
	x := 2
end
print x
`: `line 4 column 2: "x" declared but not used`,
		`
x := 1
func foo
	x := 2
	print x
end
`: `line 2 column 1: "x" declared but not used`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error(), input)
	}
}

func TestScopeErr(t *testing.T) {
	inputs := map[string]string{
		`
x := 1
x := 2
`: `line 3 column 1: redeclaration of "x"`,
		`
x := 1
x := "abc"
`: `line 3 column 1: redeclaration of "x"`,
		`
x :num
x := "abc"
`: `line 3 column 1: redeclaration of "x"`,
		`
x := "abc"
x :num
`: `line 3 column 1: redeclaration of "x"`,
		`
x :num
x :num
`: `line 3 column 1: redeclaration of "x"`,
		`
x :num
x :string
`: `line 3 column 1: redeclaration of "x"`,
		`
x :num
func x
   print "abc"
end
`: `line 2 column 1: invalid declaration of "x", already used as function name`,
		`
func x in:num
   in:string
end
`: `line 3 column 4: redeclaration of "in"`,
		`
func foo
   x := 0
   x := 0
end
`: `line 4 column 4: redeclaration of "x"`,
		`
func x
   x := 0
end
`: `line 3 column 4: invalid declaration of "x", already used as function name`,
		`
func x in:string in:string
   print in
end
`: `line 2 column 18: redeclaration of "in"`,
		`
func x x:string
   print x
end
`: `line 2 column 8: invalid declaration of "x", already used as function name`,
		`
func x x:string...
   print x
end
`: `line 2 column 8: invalid declaration of "x", already used as function name`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestIf(t *testing.T) {
	inputs := []string{
		`if true
			print "yeah"
		end`,
		`if true
			print "true"
		 else
			print "false"
		 end`,
		`if true
			print "true"
		 else if false
			print "false"
		 end`,
		`if true
			print "true"
		 else if false
			print "false"
		 else if true
			print "true true"
		 else
			print "false"
		 end`,
		`if true
			if true
				print "true true"
			else
				print "true false"
			end
		 else
			if true
				print "false true"
			else
				print "false false"
			end
		 end`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestIfErr(t *testing.T) {
	inputs := map[string]string{
		`
if true
	print "baba yaga"
`: `line 4 column 1: expected "end", got end of input`,
		`
if true
end`: `line 3 column 1: at least one statement is required here`,
		`
if
	print "baba yaga"
end`: `line 2 column 3: unexpected end of line`,
		`
if true
	print "true"
else true
	print "true"
end`: `line 4 column 6: expected end of line, found "true"`,
		`
if true
	print "true"
else if
	print "true"
end`: `line 4 column 8: unexpected end of line`,
		`
if true
	print "true"
else
   print "false"
else if false
	print "true"
end`: `line 6 column 1: unexpected input "else"`,
		`
if true
	if true
		print "true true"
else
	print "false"
end`: `line 7 column 4: expected "end", got end of input`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error(), "input: %s", input)
	}
}

func TestWhile(t *testing.T) {
	inputs := []string{
		`
while true
	print "forever"
end`,
		`
while has_more
	print "🍭"
end

two_more := true
one_more := true
func has_more:bool
	if one_more
		if two_more
			two_more = false
			return false
		else
			one_more = false
			return false
		end
	end
	return true
end
`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestWhileErr(t *testing.T) {
	inputs := map[string]string{
		`
while true
	print "forever"
`: `line 4 column 1: expected "end", got end of input`,
		`
while true
end`: "line 3 column 1: at least one statement is required here",
		`
while
	print "forever"
end`: "line 2 column 6: unexpected end of line",
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error(), "input: %s", input)
	}
}

func TestBreak(t *testing.T) {
	inputs := []string{
		`
while true
	break
end`, `
while true
	if false
		break
	end
end`, `
while true
	print "🎈"
	if true
		break
	end
	print "💣"
end`, `
func foo
	while true
		break
	end
end`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestBreakErr(t *testing.T) {
	inputs := map[string]string{
		`
while true
	break 123
end
`: "line 3 column 8: expected end of line, found 123",
		`
break
`: "line 2 column 1: break is not in a loop",
		`
if true
	break
end
`: "line 3 column 2: break is not in a loop",
		`
func x
	break
end
`: "line 3 column 2: break is not in a loop",
		`
func x
	if true
		print "foo"
	else
		break
	end
end
`: "line 6 column 3: break is not in a loop",
		`
while true
	break
	print "deadcode"
end
`: "line 4 column 2: unreachable code",
		`
while true
	if true
		break
	else
		break
	end
	print "deadcode"
end
`: "line 8 column 2: unreachable code",
		`
func a
	while true
		if true
			break
		else
			return
		end
		print "deadcode"
	end
end
`: "line 9 column 3: unreachable code",
		`
func a:num
	while true
		if true
			return 0
		else
			break
		end
		print "deadcode"
	end
end
`: "line 9 column 3: unreachable code",
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error(), "input: %s", input)
	}
}

func TestFor(t *testing.T) {
	inputs := []string{
		`
for i:= range 3
	print i
end`,
		`
for i:= range 3 5
	print i
end`,
		`
for i:= range 3 15 -1
	print i
end`,
		`
for i:= range "abc"
	print i
end`,
		`
for i:= range {}
	print i
end`,
		`
for i:= range []
	print i
end`,
		`
for i:= range []
	print i
	break
end`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestForErr(t *testing.T) {
	inputs := map[string]string{
		`
for
	print "X"
end
`: `line 2 column 4: expected "range", got end of line`,
		`
for true
	print "X"
end
`: `line 2 column 5: expected "range", got "true"`,
		`
x := 0
for x = range 5
	print "X"
end
`: `line 3 column 7: expected ":=", got "="`,
		`
for x := range 1 2 3 4
	print "X"
end
`: `line 2 column 10: range can take up to 3 num arguments, found 4`,
		`
for x := range true
	print "X"
end
`: `line 2 column 20: expected num, string, array or map after range, found bool`,
		`
for x := range 1 true
	print "X"
end
`: `line 2 column 10: range expects num type for 2nd argument, found bool`,
		`
func x
	print "func x"
end
for x := range 10
	print "x" x
end
`: `line 5 column 5: invalid declaration of "x", already used as function name`,
		`
for _ := range 10
	print "hi"
end
`: `line 2 column 5: declaration of anonymous variable "_" not allowed here`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error(), "input: %s", input)
	}
}

func TestEmptyArray(t *testing.T) {
	inputs := []string{
		`print []`,
		`print [[]]`,
		`print []+[]`,
		`print [[]]+[[]]`,
		`
		for i := range []
			print i
		end`,

		`
		arr := []
		for i := range arr
			print i
		end`,
		`
		a := []
		b := []+[]
		print a b`,
		`
		func nums n:[]num
			print n
		end

		nums []`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
		assert.Equal(t, NONE_TYPE, EMPTY_ARRAY.Sub)
	}
}

func TestEmptyMap(t *testing.T) {
	inputs := []string{
		`print {}`,
		`
m := {}

for k := range m
   print k m[k]
end`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)

		assert.Equal(t, NONE_TYPE, EMPTY_ARRAY.Sub)
	}
}

func TestFuncDefErr(t *testing.T) {
	inputs := map[string]string{
		`
func len s:string
   print "len:" s
end
`: `line 2 column 1: cannot override builtin function "len"`,
		`
func fox
   print "fox"
end

func fox
   print "fox overridden"
end
`: `line 6 column 1: redeclaration of function "fox"`,
		`
func fox _:string
   print "fox" _
end
`: `line 3 column 16: anonymous variable "_" cannot be read`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error(), "input: %s", input)
	}
}

func TestEventHandler(t *testing.T) {
	inputs := []string{
		`
on down x:num y:num
   print "pointer down:" x y
end`,
		`
on down x:num _:num
   print "pointer down x:" x
end`,
		`
on down
   print "down"
end`,
		`
on down x:num y:num
   print "pointer down:" x y
   if x > 100
      return
   end
end`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestEventHandlerErr(t *testing.T) {
	inputs := map[string]string{
		`
on down x:num y:num
   print "pointer down:" x y
`: `line 4 column 1: expected "end", got end of input`,
		`
on down:num
   print "down:" down
end
`: `line 2 column 8: expected identifier, got ":"`,
		`
on down x:num y:num
return "abc"
end
`: `line 3 column 8: expected no return value, found string`,
		`
on down2 x:num y:num
   print "down:" down
end
`: `line 2 column 4: unknown event name down2`,
		`
on down x:num
   print "pointer down:" x
end`: `line 3 column 4: wrong number of parameters expected 2, got 1`,
		`
on down x:num y:num z:num
    print x y z
end`: `line 3 column 5: wrong number of parameters expected 2, got 3`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error(), "input: %s", input)
	}
}

func TestGlobalErr(t *testing.T) {
	inputs := map[string]string{
		`
err := true
`: `line 2 column 1: redeclaration of builtin variable "err"`,
		`
errmsg := 5
`: `line 2 column 1: redeclaration of builtin variable "errmsg"`,
		`
func errmsg
   print "💣"
end
`: `line 2 column 1: cannot override builtin variable "errmsg"`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestCalledBuiltinFuncs(t *testing.T) {
	input := `print (len "ABC")`
	parser := newParser(input, testBuiltins())
	prog := parser.parse()
	assertNoParseError(t, parser, input)
	got := prog.CalledBuiltinFuncs
	sort.Strings(got)
	want := []string{"len", "print"}
	assert.Equal(t, want, got)
}

func TestBuiltinOverride(t *testing.T) {
	input := `
func len x:num
  print x
end
print (len 5)`
	parser := newParser(input, testBuiltins())
	_ = parser.parse()
	assertParseError(t, parser, input)
	gotErrs := parser.errors
	wantErrs := []string{
		`line 2 column 1: cannot override builtin function "len"`,
		`line 5 column 7: "print" takes variadic arguments of type any, found none`,
	}
	for i, err := range gotErrs {
		assert.Equal(t, err.Error(), wantErrs[i])
	}
}

func TestEmptyStringLitArg(t *testing.T) {
	input := `
fn "" 0

func fn s:string n:num
    print s n
end`
	parser := newParser(input, testBuiltins())
	parser.parse()
	assertNoParseError(t, parser, input)
}

func TestTypeAssertion(t *testing.T) {
	inputs := []string{
		`
a:any
n := a.(num)
print n`,
		`
a:any
a = 1
print a.(num)`,
		`
a:any
n := a.( num ) // whitespaces added
print n`,
		`
a:any
print a.( num ) // whitespaces added`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestArrayConcatTypingErr(t *testing.T) {
	inputs := map[string]string{
		`
b:[]num
b = [true]
`: `line 3 column 1: "b" accepts values of type []num, found []bool`,
		`
b:[]num
b = [true] + []
`: `line 3 column 1: "b" accepts values of type []num, found []bool`,
		`
b:[]num
b = [] + [true]
`: `line 3 column 1: "b" accepts values of type []num, found []bool`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestArgsWithIndex(t *testing.T) {
	inputs := []string{
		"print [1 2] [1]",
		"print [1 2][1]",
		"print {} []",
		"print [] []",
		"print [] {}",
		"print {} {}",
		`print {a:1}["a"]`,
		`print {a:1} ["a"]`,
		`
func fn:{}num
	return {a:1}
end
print (fn)["a"]
print (fn) ["a"]
`,
		`
func fn:string
	return "abc"
end
print (fn)[1]
print (fn) [1]
`,
		`a:any
a = [1 2]
print a.([]num) [1]
print a.([]num)[1]
`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestLateCompositeLiteralTyping(t *testing.T) {
	inputs := []string{
		`
a:[]any
m:{}any
a = [1 2 3]
m = {a:true b:false}
print a m
`, `
func fn m:{}any a:[]any
    print m a
end
fn {} []
fn {a:1} [true] // {}num []bool`,
		`has {} "b"`,
		`has {x:2} "b"`,
		`has {x:{b:2}} "b"`,
		`
func fn a:[]any...
	 print a
end
fn []
fn [1 2]
fn [] []
fn [1] [true]
`, `
func fn m:{}any...
	 print m
end
fn {}
fn {a:1 b:2}
fn {} {}
fn {a:1} {b:true}
`, `
func fnm:{}any
	 return {a:{b:2}}
end
`, `
func fna:[]any
	 return [true]
end
`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestArrayTypeCombiningNoErr(t *testing.T) {
	inputs := []string{
		`
n := 0
na := [1]
print [na [true]] // []any
print [[] []] // [][]any
print [[] [2]] // [][]num
print [["string"] [2]] // [][]any
print [[n] [2]] // [][]num
print [na [2]] // [][]num
print [na na] // [][]num
print [[n] [true]] // [][]any
print [na true] // []any
`,
		`
arr := [1]
print [[arr] [[true]]] // [][]any
print [[[1]] [[true]]] // [][][]any
print [[arr] [[1]]] // [][][]num
`,
		`
a:[]any
n := 2
a = [1 2 n]
print "[1 2 n]" [1 2 n] // [1 2 n] []num
print a a // [1 2 2] []any
`,
		`
ba:[]bool
foo := [[1] ba]
print foo
foo2 := [[true] ba] // []any
print foo2 // [][]bool
		`,
	}

	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		parser.parse()
		assertNoParseError(t, parser, input)
	}
}

func TestLateCompositeLiteralTypingErr(t *testing.T) {
	inputs := map[string]string{
		`
has ["a"] "a"`: `line 2 column 5: "has" takes 1st argument of type {}, found []string`,
		`
a:[]any
b := [1 2]
a = [1 2] // ok, literal becomes []any
a = b  // fail
`: `line 5 column 1: "a" accepts values of type []any, found []num`,
		`
b := [1 2]
b = [] + b + [true]
`: `line 3 column 12: mismatched type for +: []num, []bool`,
		`
a:[]any
b:[]num
b = [1 2 3]
a = [] + b
`: `line 5 column 1: "a" accepts values of type []any, found []num`,
		`
func fn m:{}any...
	 print m
end
m:{}num
fn m
`: `line 6 column 4: "fn" takes variadic arguments of type {}any, found {}num`,
		`
func fn:{}any
	m := {a:1}
	return m
end
`: `line 4 column 9: expected return value of type {}any, found {}num`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestMapLitErr(t *testing.T) {
	inputs := map[string]string{
		`print {a:1{b:2}}`:    `line 1 column 11: expected map key, found "{"`,
		`print {"x": true}`:   `line 1 column 8: expected map key, found "x"`,
		`print {"end": true}`: `line 1 column 8: expected map key, found "end"`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestInferredVarType(t *testing.T) {
	input := `
arr := [[]]
print arr`
	want := `
arr=[[]]
print(any(arr))
`
	parser := newParser(input, testBuiltins())
	prog := parser.parse()
	assertNoParseError(t, parser, input)
	got := prog.String()
	assert.Equal(t, want, got)
	stmt, ok := prog.Statements[1].(*InferredDeclStmt)
	assert.Equal(t, true, ok, "%v %T", stmt, stmt)

	want = "[][]any"
	got = stmt.Decl.Type().String()
	assert.Equal(t, want, got)

	want = "[][]any"
	got = stmt.Decl.Value.Type().String()
	assert.Equal(t, want, got)

	arrayLit, ok := stmt.Decl.Value.(*ArrayLiteral)
	assert.Equal(t, true, ok, "%v %T", arrayLit, arrayLit)

	want = "[]any"
	got = arrayLit.Elements[0].Type().String()
	assert.Equal(t, want, got)
}

func TestWrapArrayNoError(t *testing.T) {
	inputs := []string{
		`
a:[]any
n := 2
a = [1 2 n]
print "[1 2 n]" (typeof [1 2 n])
print a (typeof a)`,
	}
	for _, input := range inputs {
		parser := newParser(input, testBuiltins())
		assertNoParseError(t, parser, input)
	}
}

func TestAnyWrapArray(t *testing.T) {
	input := `
x:[]any
x = [1 2 true]
print x
`
	want := `
x=[]
x = [any(1), any(2), any(true)]
print(any(x))
`
	parser := newParser(input, testBuiltins())
	prog := parser.parse()
	assertNoParseError(t, parser, input)
	got := prog.String()
	assert.Equal(t, want, got)

	stmt, ok := prog.Statements[2].(*AssignmentStmt)
	assert.Equal(t, true, ok, "%v %T", stmt, stmt)

	arrayLit, ok := stmt.Value.(*ArrayLiteral)
	assert.Equal(t, true, ok, "%v %T", arrayLit, arrayLit)

	want = "any"
	got = arrayLit.Elements[0].Type().String()
	assert.Equal(t, want, got)
}

func TestAnyWrapParam(t *testing.T) {
	input := `
f []
f [[]]
func f a:any
    print a
end`
	want := `
f(any([]))
f(any([[]]))
f(a){
print(a)
}

`
	parser := newParser(input, testBuiltins())
	prog := parser.parse()
	assertNoParseError(t, parser, input)
	got := prog.String()
	assert.Equal(t, want, got)

	stmt, ok := prog.Statements[1].(*FuncCallStmt)
	assert.Equal(t, true, ok, "%v %T", stmt, stmt)

	arrayLit, ok := stmt.FuncCall.Arguments[0].(*Any).Value.(*ArrayLiteral)
	assert.Equal(t, true, ok, "%v %T", stmt.FuncCall.Arguments[0], stmt.FuncCall.Arguments[0].(*Any).Value)

	want = "[]any"
	got = arrayLit.Type().String()
	assert.Equal(t, want, got)

	stmt, ok = prog.Statements[2].(*FuncCallStmt)
	assert.Equal(t, true, ok, "%v %T", stmt, stmt)

	arrayLit, ok = stmt.FuncCall.Arguments[0].(*Any).Value.(*ArrayLiteral)
	assert.Equal(t, true, ok, "%v %T", stmt.FuncCall.Arguments[0], stmt.FuncCall.Arguments[0].(*Any).Value)

	want = "[][]any"
	got = arrayLit.Type().String()
	assert.Equal(t, want, got)

	arrayLit, ok = arrayLit.Elements[0].(*ArrayLiteral)
	assert.Equal(t, true, ok, "%v %T", stmt.FuncCall.Arguments[0], stmt.FuncCall.Arguments[0].(*Any).Value)

	want = "[]any"
	got = arrayLit.Type().String()
	assert.Equal(t, want, got)
}

func TestBadArrayParamType(t *testing.T) {
	inputs := map[string]string{
		`
func fn a:[][]any
    print a
end
a:[]num
fn [a]
`: `line 6 column 4: "fn" takes 1st argument of type [][]any, found [][]num`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestBadMapParamType(t *testing.T) {
	inputs := map[string]string{
		`
func fn a:{}{}any
    print a
end
a:{}num
fn {x:a}
`: `line 6 column 4: "fn" takes 1st argument of type {}{}any, found {}{}num`,
	}
	for input, wantErr := range inputs {
		parser := newParser(input, testBuiltins())
		_ = parser.parse()
		assertParseError(t, parser, input)
		gotErr := parser.errors.Truncate(1)
		assert.Equal(t, wantErr, gotErr.Error())
	}
}

func TestBadFuncSignature(t *testing.T) {
	input := `
left_pos := {x:0 y:50}

func draw_paddle paddle:map
    print paddle.x paddle.y-10
end

draw_paddle left_pos
`
	_, err := Parse(input, testBuiltins())
	parseErrors := &Errors{}
	assert.Equal(t, true, errors.As(err, parseErrors))

	got := (*parseErrors)[0].Error()
	want := `line 4 column 18: invalid type declaration for "paddle"`
	assert.Equal(t, want, got)
}

func TestDemo(t *testing.T) {
	input := `
move 10 10
line 20 20

x := 12
print "x:" x
if x > 10
    print "🍦 big x"
end`
	parser := newParser(input, testBuiltins())
	got := parser.parse()
	assertParseError(t, parser, input)
	gotErr := parser.errors.Truncate(1)
	assert.Equal(t, `line 2 column 1: unknown function "move"`, gotErr.Error())
	assert.Equal(t, `line 3 column 1: unknown function "line"`, parser.errors[1].Error())
	want := `

x=12
print(any("x:"), any(x))
if ((x>10)) {
print(any("🍦 big x"))
}
`
	assert.Equal(t, want, got.String())
}

func assertParseError(t *testing.T, parser *parser, input string) {
	t.Helper()
	assert.Equal(t, true, len(parser.errors) > 0, "expected parser errors, got none: input: %s\n", input)
}

func assertNoParseError(t *testing.T, parser *parser, input string) {
	t.Helper()
	assert.Equal(t, 0, len(parser.errors), "Unexpected parser error\n input: %s\nerrors:\n%s", input, parser.errors)
}

func testBuiltins() Builtins {
	funcs := map[string]*FuncDefStmt{
		"print": {
			Name:          "print",
			VariadicParam: &Var{Name: "a", T: ANY_TYPE},
			ReturnType:    NONE_TYPE,
		},
		"len": {
			Name:       "len",
			Params:     []*Var{{Name: "a", T: ANY_TYPE}},
			ReturnType: NUM_TYPE,
		},
		"has": {
			Name: "has",
			Params: []*Var{
				{Name: "map", T: GENERIC_MAP},
				{Name: "key", T: STRING_TYPE},
			},
			ReturnType: NONE_TYPE,
		},
		"join": {
			Name: "join",
			Params: []*Var{
				{Name: "arr", T: GENERIC_MAP},
				{Name: "sep", T: STRING_TYPE},
			},
			ReturnType: STRING_TYPE,
		},
	}
	eventHandlers := map[string]*EventHandlerStmt{
		"down": {
			Name: "down",
			Params: []*Var{
				{Name: "x", T: NUM_TYPE},
				{Name: "y", T: NUM_TYPE},
			},
		},
	}
	globals := map[string]*Var{
		"err":    {Name: "err", T: BOOL_TYPE},
		"errmsg": {Name: "errmsg", T: STRING_TYPE},
	}

	return Builtins{
		Funcs:         funcs,
		EventHandlers: eventHandlers,
		Globals:       globals,
	}
}
