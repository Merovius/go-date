// Copyright 2024 Axel Wagner.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package date

import (
	"strings"
	"testing"
	"time"

	"gonih.org/set"
)

var layouts = []string{
	Layout,
	RFC822,
	RFC1123,
	RFC3339,
}

// FuzzParseLayout generates layouts to check that [parseLayout] does not
// panic.
func FuzzParseLayout(f *testing.F) {
	f.Add(time.Layout)
	f.Add(time.ANSIC)
	f.Add(time.UnixDate)
	f.Add(time.RubyDate)
	f.Add(time.RFC822)
	f.Add(time.RFC822Z)
	f.Add(time.RFC850)
	f.Add(time.RFC1123)
	f.Add(time.RFC1123Z)
	f.Add(time.RFC3339)
	f.Add(time.RFC3339Nano)
	f.Add(time.Kitchen)
	f.Add(time.Stamp)
	f.Add(time.StampMilli)
	f.Add(time.StampMicro)
	f.Add(time.StampNano)
	f.Add(time.DateTime)
	f.Add(time.DateOnly)
	f.Add(time.TimeOnly)
	for _, l := range layouts {
		f.Add(l)
	}
	f.Fuzz(func(t *testing.T, s string) {
		parseLayout(s)
	})
}

// FuzzFormat generates layouts and Date values to check that [Date.Format]
// does not panic.
func FuzzFormat(f *testing.F) {
	d := int(Of(2023, 10, 25))
	f.Add(time.Layout, d)
	f.Add(time.ANSIC, d)
	f.Add(time.UnixDate, d)
	f.Add(time.RubyDate, d)
	f.Add(time.RFC822, d)
	f.Add(time.RFC822Z, d)
	f.Add(time.RFC850, d)
	f.Add(time.RFC1123, d)
	f.Add(time.RFC1123Z, d)
	f.Add(time.RFC3339, d)
	f.Add(time.RFC3339Nano, d)
	f.Add(time.Kitchen, d)
	f.Add(time.Stamp, d)
	f.Add(time.StampMilli, d)
	f.Add(time.StampMicro, d)
	f.Add(time.StampNano, d)
	f.Add(time.DateTime, d)
	f.Add(time.DateOnly, d)
	f.Add(time.TimeOnly, d)
	for _, l := range layouts {
		f.Add(l, d)
	}
	f.Fuzz(func(t *testing.T, layout string, date int) {
		if date < 0 {
			return
		}
		Date(date).Format(layout)
	})
}

// FuzzFormatCompat generates layouts and values to compare the formatting of
// [time] to our implementation.
//
// As [time] supports more format specifiers, which would create expected
// deviations in behavior, the fuzzing target uses a binary representation for
// layouts which can more easily be filtered for such layout strings.
func FuzzFormatCompat(f *testing.F) {
	f.Fuzz(func(t *testing.T, progBytes []byte, date int) {
		layout, ok := decodeProg(progBytes)
		if !ok {
			return
		}
		d := Date(date)
		got, want := d.Format(layout), d.Time(8, 0, 0, 0, time.UTC).Format(layout)
		if got != want {
			t.Fatalf("%#v.Format(%q) returned different string from (time.Time).Format: got %q, want %q", d, layout, got, want)
		}
	})
}

// TestFormat checks that formatting works as expected.
func TestFormat(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		date   Date
		layout string
		want   string
	}{
		{Of(2006, 1, 2), RFC822, RFC822},
		{Of(2006, 1, 2), RFC1123, RFC1123},
		{Of(2006, 1, 2), RFC1123, RFC1123},
		{Of(2023, 10, 25), RFC822, "25 Oct 23"},
		{Of(2023, 10, 25), RFC1123, "25 Oct 2023"},
		{Of(2023, 10, 25), RFC3339, "2023-10-25"},
		{Of(2006, 1, 2), time.ANSIC, strings.ReplaceAll(time.ANSIC, "_", " ")},
		{Of(2023, 10, 25), "_2006-01-02", "_2023-10-25"},
		{Of(-2023, 10, 25), RFC3339, "-2023-10-25"},
		{Of(-2003, 10, 25), RFC822, "25 Oct 03"},
		{Of(-2023, 10, 25), RFC822, "25 Oct 23"},
		{Of(2023, 10, 25), "January 2", "October 25"},
		{Of(2023, 1, 23), RFC3339, "2023-01-23"},
		{Of(2023, 10, 25), "Monday", "Wednesday"},
		{Of(2023, 10, 25), "__2", "298"},
		{Of(2023, 3, 2), "__2", " 61"},
		{Of(2023, 1, 9), "__2", "  9"},
		{Of(2023, 10, 25), "002", "298"},
		{Of(2023, 3, 2), "002", "061"},
		{Of(2023, 1, 9), "002", "009"},
		{Of(2, 1, 1), "2006", "0002"},
		{Of(23, 1, 1), "2006", "0023"},
		{Of(420, 1, 1), "2006", "0420"},
	}
	for _, tc := range tcs {
		if got := tc.date.Format(tc.layout); got != tc.want {
			t.Errorf("%#v.Format(%q) = %q, want %q", tc.date, tc.layout, got, tc.want)
		}
	}
}

// FuzzParse generates layouts and values to check that Parse does not panic.
func FuzzParse(f *testing.F) {
	f.Fuzz(func(t *testing.T, layout, value string) {
		Parse(layout, value)
	})
}

// FuzzParseCompat generates layouts and values to compare the parsing of
// [time] to our implementation.
//
// As [time] supports more format specifiers, which would create expected
// deviations in behavior, the fuzzing target uses a binary representation for
// layouts which can more easily be filtered for such layout strings.
func FuzzParseCompat(f *testing.F) {
	f.Fuzz(func(t *testing.T, progBytes []byte, value string) {
		layout, ok := decodeProg(progBytes)
		if !ok {
			return
		}
		d, errD := Parse(layout, value)
		T, errT := time.Parse(layout, value)
		if (errD == nil) != (errT == nil) {
			t.Fatalf("Parse(%q, %q) returned different error from time.Parse: got %v, want %v", layout, value, errD, errT)
		}
		td := Of(T.Date())
		if d != td {
			t.Fatalf("Parse(%q, %q) returned different date than time.Parse: got %#v, want %#v", layout, value, d, td)
		}
	})
}

func TestParse(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		layout string
		value  string
		want   Date
	}{
		{RFC822, RFC822, Of(2006, 1, 2)},
		{RFC1123, RFC1123, Of(2006, 1, 2)},
		{RFC3339, RFC3339, Of(2006, 1, 2)},
		{RFC3339, "2023-10-31", Of(2023, 10, 31)},
		{RFC3339, "2023 10 31", 0},
		{"", "", Of(0, 1, 1)},
		{"06", "1", 0},
		{"06", "foo", 0},
		{"06", "69", Of(1969, 01, 01)},
		{"06", "23", Of(2023, 01, 01)},
		{"2006", "1", 0},
		{"2006", "foobar", 0},
		{"Jan", "F", 0},
		{"Jan", "foo", 0},
		{"Jan", "Feb", Of(0, 2, 1)},
		{"Jan", "fEb", Of(0, 2, 1)},
		{"January", "Feb", 0},
		{"January", "Aviary", 0},
		{"January", "February", Of(0, 2, 1)},
		{"January", "FeBrUaRy", Of(0, 2, 1)},
		{"1", "", 0},
		{"1", "x", 0},
		{"1", "2", Of(0, 2, 1)},
		{"1", "12", Of(0, 12, 1)},
		{"1", "02", Of(0, 2, 1)},
		{"1", "13", 0},
		{"1", "0", 0},
		{"01", "x", 0},
		{"01", "xx", 0},
		{"01", "2", 0},
		{"01", "12", Of(0, 12, 1)},
		{"01", "02", Of(0, 2, 1)},
		{"Mon", "T", 0},
		{"Mon", "foo", 0},
		{"Mon", "Tue", Of(0, 1, 1)}, // Weekday names are ignored except for parsing
		{"Mon", "TuE", Of(0, 1, 1)},
		{"Monday", "T", 0},
		{"Monday", "foobar", 0},
		{"Monday", "Tuesday", Of(0, 1, 1)},
		{"Monday", "TuEsDaY", Of(0, 1, 1)},
		{"2", "", 0},
		{"2", "x", 0},
		{"2", "3", Of(0, 1, 3)},
		{"2", "03", Of(0, 1, 3)},
		{"2", "31", Of(0, 1, 31)},
		{"2", "32", 0},
		{"2", "0", 0},
		{"02", "x", 0},
		{"02", "xx", 0},
		{"02", "3", 0},
		{"02", "03", Of(0, 1, 3)},
		{"02", "31", Of(0, 1, 31)},
		{"02", "32", 0},
		{"_2", "x", 0},
		{"_2", "xx", 0},
		{"_2", "3", Of(0, 1, 3)},
		{"_2", " 3", Of(0, 1, 3)},
		{"_2", "  3", 0},
		{"_2", "03", Of(0, 1, 3)},
		{"_2", "31", Of(0, 1, 31)},
		{"_2", "32", 0},
		{"002", "x", 0},
		{"002", "xx", 0},
		{"002", "3", 0},
		{"002", "03", 0},
		{"002", "003", Of(0, 1, 3)},
		{"002", "050", Of(0, 2, 19)},
		{"002", "298", Of(0, 10, 24)},
		{"__2", "x", 0},
		{"__2", "xx", 0},
		{"__2", "3", Of(0, 1, 3)},
		{"__2", " 3", Of(0, 1, 3)},
		{"__2", "  3", Of(0, 1, 3)},
		{"__2", "   3", 0},
		{"__2", "03", Of(0, 1, 3)},
		{"__2", " 03", Of(0, 1, 3)},
		{"__2", "  03", Of(0, 1, 3)},  // consistent with time.Parse
		{"__2", "  003", Of(0, 1, 3)}, // consistent with time.Parse
		{"__2", "   03", 0},
		{"__2", "003", Of(0, 1, 3)},
		{"__2", "050", Of(0, 2, 19)},
		{"__2", "298", Of(0, 10, 24)},
		{RFC3339, RFC3339 + "foo", 0},
		{"2006-01-02 002", "2023-10-25 100", 0},
		{"2006-01-02 002", "2023-10-25 300", 0},
		{"2006-01-02 002", "2023-10-25 298", Of(2023, 10, 25)},
		{"2006-01-02 002", "2024-10-25 299", Of(2024, 10, 25)},
		{"002", "0", 0},
		{"2006 __2", "2023 366", 0},
		{"2006 __2", "2024 366", Of(2024, 12, 31)},
		{"2006 __2", "2023 60", Of(2023, 03, 01)},
		{"2006 __2", "2024 60", Of(2024, 02, 29)},
		{"   2006", " 2023", Of(2023, 1, 1)},
	}
	for _, tc := range tcs {
		got, err := Parse(tc.layout, tc.value)
		gotT, errT := time.Parse(tc.layout, tc.value)
		if (err == nil) != (errT == nil) {
			t.Errorf("Parse(%q, %q) returned different error from time.Parse: got %v, want %v", tc.layout, tc.value, err, errT)
		}
		if err != nil {
			continue
		}
		if got != tc.want {
			t.Errorf("Parse(%q, %q) = %#v, want %#v", tc.layout, tc.value, got, tc.want)
		}
		if want := Of(gotT.Date()); got != want {
			t.Errorf("Parse(%q, %q) returned different date than time.Parse: got %#v, want %#v", tc.layout, tc.value, got, want)
		}
	}
}

// TestParseZeroAllocs checks that calling Parse does not escape its argument
// and does not allocate, in the happy path.
func TestParseZeroAllocs(t *testing.T) {
	const want = 0.0
	const layout = "Monday, 2006-01-02 002"
	const value = "Thursday, 2023-11-02 306"

	got := testing.AllocsPerRun(10000, parseHappy)
	if got != want {
		t.Fatalf("Parse allocates %v times, want %v", got, want)
	}
}

// BenchmarkParseHappy benchmarks (and counts allocations) of Parse in the
// happy path.
func BenchmarkParseHappy(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parseHappy()
	}
}

func parseHappy() {
	const layout = "Monday, 2006-01-02 002"
	const value = "Thursday, 2023-11-02 306"
	b := make([]byte, len(value))
	copy(b, value)
	_, _ = Parse(layout, string(b))
}

// decodeProg tries to parse b into a slice of inst for use in fuzzing, with a
// simple format. It validates that no literal instructions contain any format
// specifiers supported by package time but not by this package.
//
// The format consists of a sequence of encoded inst. The first byte is the
// fmtOp value (and must be in range). If the fmtOp is fmtLiteral, it must be
// followed by the literal, prefixed with a one-byte length.
func decodeProg(b []byte) (string, bool) {
	layout := new(strings.Builder)
	for len(b) > 0 {
		var (
			op  fmtOp
			n   int
			lit string
		)
		op, b = fmtOp(b[0]), b[1:]
		if op < 0 || op >= opInvalid {
			return "", false
		}
		if op != opLiteral {
			layout.WriteString(op.String())
			continue
		}
		if len(b) == 0 {
			return "", false
		}
		n, b = int(b[0]), b[1:]
		if n > len(b) {
			return "", false
		}
		lit, b = string(b[:n]), b[n:]
		for s := range timeSpecs {
			if strings.Contains(lit, s) {
				return "", false
			}
		}
		layout.WriteString(lit)
	}
	return layout.String(), true
}

// timeSpecs are format specifiers supported by package time that are not used by date.
var timeSpecs = set.Make("15", "3", "03", "PM", "4", "04", "5", "05", "-0700", "-07:00", "-07", "-070000", "-07:00:00", "Z0700", "Z07:00", "Z07", "Z070000", "Z07:00:00", ".0", ",0", ".9", ",9")
