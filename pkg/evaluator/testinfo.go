package evaluator

import "strconv"

// TestReporting represents an enumeration of reporting styles.
type TestReporting int

// Test reporting styles: terse, verbose or none.
const (
	TerseReporting TestReporting = iota
	VerboseReporting
	NoReporting
)

// TestInfo contains flags for test runs, e.g. FailFast and testResult
// information, e.g. total count.
type TestInfo struct {
	FailFast  bool
	Reporting TestReporting

	errors []error
	total  int
}

// FailCount returns the number of failed assertions.
func (t *TestInfo) FailCount() int {
	return len(t.errors)
}

// SuccessCount returns the number of successful assertions.
func (t *TestInfo) SuccessCount() int {
	return t.total - len(t.errors)
}

// TotalCount returns the total number of assertions executed.
func (t *TestInfo) TotalCount() int {
	return t.total
}

// Report prints a summary of the test results.
func (t *TestInfo) Report(printFn func(string)) {
	if t.Reporting == NoReporting || t.total == 0 {
		return
	}
	var msg string
	switch {
	case t.FailCount() > 0:
		msg = "❌ " + strconv.Itoa(t.FailCount()) + " failed" + "\n" +
			"   " + strconv.Itoa(t.SuccessCount()) + " passed\n"
	case t.SuccessCount() != 1:
		msg = "✅ " + strconv.Itoa(t.SuccessCount()) + " passed assertions\n"
	default:
		msg = "✅ " + strconv.Itoa(t.SuccessCount()) + " passed assertion\n"
	}
	printFn(msg)
}
