// Copyright 2024 Axel Wagner.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package date

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gonih.org/date/internal/cache"
)

// These are predefined layouts for use in [Date.Format] and [Parse]. The
// reference date used in these layouts is the specific date:
//
//	January 2, 2006
//
// That value is recorded as the constant named [Layout], listed below. The date
// is chosen for compatibility with package [time].
//
// The format specification works the same as [time.Layout], except that format
// specifiers related to time and timezones are treated as literals and
// otherwise ignored. Specifically, the recognized components are
//
//	Year: "2006" "06"
//	Month: "Jan" "January" "01" "1"
//	Day of the week: "Mon" "Monday"
//	Day of the month: "2" "_2", "02"
//	Day of the year: "__2" "002"
const (
	Layout  = "01/02 '06" // The reference date, in numerical order
	RFC822  = "02 Jan 06"
	RFC1123 = "02 Jan 2006"
	RFC3339 = "2006-01-02"
)

var longDayNames = []string{
	"Sunday",
	"Monday",
	"Tuesday",
	"Wednesday",
	"Thursday",
	"Friday",
	"Saturday",
}

var shortDayNames = []string{
	"Sun",
	"Mon",
	"Tue",
	"Wed",
	"Thu",
	"Fri",
	"Sat",
}

var shortMonthNames = []string{
	"Jan",
	"Feb",
	"Mar",
	"Apr",
	"May",
	"Jun",
	"Jul",
	"Aug",
	"Sep",
	"Oct",
	"Nov",
	"Dec",
}

var longMonthNames = []string{
	"January",
	"February",
	"March",
	"April",
	"May",
	"June",
	"July",
	"August",
	"September",
	"October",
	"November",
	"December",
}

// inst is a single component of a layout string, either a literal string, or a
// formatting operator.
type inst struct {
	op  fmtOp
	lit string
}

// String implements fmt.Stringer, for debugging
func (i inst) String() string {
	if i.op == opLiteral {
		return i.lit
	}
	return i.op.String()
}

// fmtOp is a formatting operator.
type fmtOp int

const (
	opLiteral fmtOp = iota

	// Sorted by parsing preference, do not re-order!
	opLongMonth
	opMonth
	opLongWeekDay
	opWeekDay
	opZeroYearDay
	opZeroMonth
	opZeroDay
	opYear
	opNumMonth
	opLongYear
	opDay
	opUnderLongYear // package time treats this as "_"+opLongYear, but it is simpler to just handle it with an extra opcode
	opUnderDay
	opUnderYearDay

	opInvalid
)

// String implements fmt.Stringer. Except for opLiteral, it returns the layout
// component of the operator.
func (op fmtOp) String() string {
	switch op {
	case opLiteral:
		return "<literal>"
	case opLongMonth:
		return "January"
	case opMonth:
		return "Jan"
	case opLongWeekDay:
		return "Monday"
	case opWeekDay:
		return "Mon"
	case opZeroYearDay:
		return "002"
	case opZeroMonth:
		return "01"
	case opZeroDay:
		return "02"
	case opYear:
		return "06"
	case opNumMonth:
		return "1"
	case opLongYear:
		return "2006"
	case opDay:
		return "2"
	case opUnderLongYear:
		return "_2006"
	case opUnderDay:
		return "_2"
	case opUnderYearDay:
		return "__2"
	}
	panic("invalid fmtOp")
}

// endsWord returns whether op must be a full word, that is must not be
// followed by a lower-case letter.
func (op fmtOp) endsWord() bool {
	return op == opMonth || op == opWeekDay
}

// memoize compiled layout strings.
var memo cache.Cache[string, []inst]

// parseLayout parses layout into a set of instructions to parse or format
// according to it.
func parseLayout(layout string) []inst {
	var prog []inst
	for len(layout) > 0 {
		prefix, op, suffix := nextOp(layout)
		if prefix != "" {
			prog = append(prog, inst{lit: prefix})
		}
		if op != opLiteral {
			prog = append(prog, inst{op: op})
		}
		layout = suffix
	}
	return prog
}

// nextOp decomposes layout into the next operator, a literal prefix and the
// rest of the layout.
func nextOp(layout string) (prefix string, op fmtOp, suffix string) {
	for i := 0; i < len(layout); i++ {
		for op := opLongMonth; op < opInvalid; op++ {
			suffix, ok := strings.CutPrefix(layout[i:], op.String())
			if !ok {
				continue
			}
			if op.endsWord() && startsWithLowerCase(suffix) {
				continue
			}
			return layout[:i], op, suffix
		}
	}
	return layout, opLiteral, ""
}

// startsWithLowerCase reports whether the string has a lower-case letter at
// the beginning. Its purpose is to prevent matching strings like "Month" when
// looking for "Mon".
func startsWithLowerCase(s string) bool {
	return len(s) > 0 && 'a' <= s[0] && s[0] <= 'z'
}

// Format returns a textual representation of the date value formatted
// according to the layout defined by the argument. See the documentation for
// the constant called Layout to see how to represent the layout format.
func (d Date) Format(layout string) string {
	const bufSize = 64
	var b []byte
	max := len(layout) + 10
	if max < bufSize {
		var buf [64]byte
		b = buf[:0]
	} else {
		b = make([]byte, 0, max)
	}
	return string(d.AppendFormat(b, layout))
}

// AppendFormat is like Format but appends the textual representation to b and
// returns the extended buffer.
func (d Date) AppendFormat(b []byte, layout string) []byte {
	year, month, day, yday := absDate(d.abs(), true)
	yday++

	prog := memo.Get(layout, parseLayout)

	for _, i := range prog {
		switch i.op {
		case opLiteral:
			b = append(b, i.lit...)
		case opYear:
			y := int64(year) % 100
			if y < 0 {
				y = -y
			}
			if y < 10 {
				b = append(b, '0')
			}
			b = strconv.AppendInt(b, y, 10)
		case opUnderLongYear:
			b = append(b, '_')
			fallthrough
		case opLongYear:
			y := year
			if y < 0 {
				b = append(b, '-')
				y = -y
			}
			if y < 1000 {
				b = append(b, '0')
			}
			if y < 100 {
				b = append(b, '0')
			}
			if y < 10 {
				b = append(b, '0')
			}
			b = strconv.AppendInt(b, int64(y), 10)
		case opMonth:
			b = append(b, month.String()[:3]...)
		case opLongMonth:
			b = append(b, month.String()...)
		case opNumMonth:
			b = strconv.AppendInt(b, int64(month), 10)
		case opZeroMonth:
			if month < 10 {
				b = append(b, '0')
			}
			b = strconv.AppendInt(b, int64(month), 10)
		case opWeekDay:
			b = append(b, d.Weekday().String()[:3]...)
		case opLongWeekDay:
			b = append(b, d.Weekday().String()...)
		case opDay:
			b = strconv.AppendInt(b, int64(day), 10)
		case opUnderDay:
			if day < 10 {
				b = append(b, ' ')
			}
			b = strconv.AppendInt(b, int64(day), 10)
		case opZeroDay:
			if day < 10 {
				b = append(b, '0')
			}
			b = strconv.AppendInt(b, int64(day), 10)
		case opUnderYearDay:
			if yday < 100 {
				b = append(b, ' ')
				if yday < 10 {
					b = append(b, ' ')
				}
			}
			b = strconv.AppendInt(b, int64(yday), 10)
		case opZeroYearDay:
			if yday < 100 {
				b = append(b, '0')
				if yday < 10 {
					b = append(b, '0')
				}
			}
			b = strconv.AppendInt(b, int64(yday), 10)
		default:
			panic(errors.New("invalid inst " + i.String()))
		}
	}
	return b
}

// Parse parses a formatted string and returns the date value it represents.
// See the documentation for the constant called Layout to see how to represent
// the format. The second argument must be parseable using the format string
// (layout) provided as the first argument.
//
// Elements omitted from the layout are assumed to be zero or, when zero is
// impossible, one. Years must be in the range 0000…9999. The day of the week
// is checked for syntax but is otherwise ignored.
//
// For layouts specifying the two-digit year 06, a value NN >= 69 will be
// treated as 19NN and a value NN < 69 will be treated as 20NN.
func Parse(layout, value string) (Date, error) {
	p := newParser(value)
	var (
		// kept around for error reporting
		alayout, avalue = layout, value
		year            int
		month           int = -1
		day             int = -1
		yday            int = -1
	)

	prog := memo.Get(layout, parseLayout)

	// Execute the parsing instructions
	for _, i := range prog {
		p.setInst(i)
		switch i.op {
		case opLiteral:
			p.accept(i.lit)
		case opYear:
			year = p.atoi(2)
			if year >= 69 { // Unix time starts Dec 31 1969 in some time zones
				year += 1900
			} else {
				year += 2000
			}
		case opUnderLongYear:
			p.accept("_")
			fallthrough
		case opLongYear:
			p.peekDigit()
			year = p.atoi(4)
		case opMonth:
			month = p.lookup(shortMonthNames) + 1
		case opLongMonth:
			month = p.lookup(longMonthNames) + 1
		case opNumMonth, opZeroMonth:
			month = p.num(i.op == opZeroMonth)
			if month <= 0 || 12 < month {
				return 0, p.err(alayout, avalue, "month out of range")
			}
		case opWeekDay:
			// ignore weekday, except for parsing
			p.lookup(shortDayNames)
		case opLongWeekDay:
			// ignore weekday, except for parsing
			p.lookup(longDayNames)
		case opUnderDay:
			p.skipByte(' ')
			fallthrough
		case opDay, opZeroDay:
			day = p.num(i.op == opZeroDay)
		case opUnderYearDay:
			p.skipByte(' ')
			p.skipByte(' ')
			fallthrough
		case opZeroYearDay:
			yday = p.num3(i.op == opZeroYearDay)
		default:
			panic(errors.New("invalid inst " + i.String()))
		}
		if p.hasErr {
			return 0, p.err(alayout, avalue, "")
		}
	}
	if len(p.value) > 0 {
		return 0, p.err(alayout, avalue, "extra text: "+strconv.Quote(p.value))
	}
	p.finish()

	// Validate the parsed date
	if yday >= 0 {
		var (
			d int
			m int
		)
		if isLeap(year) {
			if yday == 31+29 {
				m = int(time.February)
				d = 29
			} else if yday > 31+29 {
				yday--
			}
		}
		if yday < 1 || yday > 365 {
			return 0, p.err(alayout, avalue, "day-of-year out of range")
		}
		if m == 0 {
			m = (yday-1)/31 + 1
			if int(daysBefore[m]) < yday {
				m++
			}
			d = yday - int(daysBefore[m-1])
		}
		// If month, day already seen, yday's m, d must match.
		// Otherwise, set them from m, d.
		if month >= 0 && month != m {
			return 0, p.err(alayout, avalue, "day-of-year does not match month")
		}
		month = m
		if day >= 0 && day != d {
			return 0, p.err(alayout, avalue, "day-of-year does not match day")
		}
		day = d
	} else {
		if month < 0 {
			month = int(time.January)
		}
		if day < 0 {
			day = 1
		}
	}
	// Validate the day of the month.
	if day < 1 || day > daysIn(time.Month(month), year) {
		return 0, p.err(alayout, avalue, "day out of range")
	}
	return Of(year, time.Month(month), day), nil
}

// match reports whether s1 and s2 match ignoring case.
// It is assumed s1 and s2 are the same length.
func match(s1, s2 string) bool {
	for i := 0; i < len(s1); i++ {
		c1 := s1[i]
		c2 := s2[i]
		if c1 != c2 {
			// Switch to lower-case; 'a'-'A' is known to be a single bit.
			c1 |= 'a' - 'A'
			c2 |= 'a' - 'A'
			if c1 != c2 || c1 < 'a' || c1 > 'z' {
				return false
			}
		}
	}
	return true
}

func isDigit(s string, i int) bool {
	if len(s) <= i {
		return false
	}
	return '0' <= s[i] && s[i] <= '9'
}

type parser struct {
	inst   inst
	hasErr bool
	value  string
	valEl  string
	errMsg string
}

func newParser(value string) *parser {
	return &parser{
		value: value,
	}
}

// setInst sets the current instruction and input offset for error reporting.
func (p *parser) setInst(i inst) {
	p.inst = i
	p.valEl = p.value
}

// finish signals that parsing is finished and the parser is only being kept
// around for error reporting.
func (p *parser) finish() {
	p.inst = inst{op: opInvalid}
	p.valEl = ""
}

// parseFailed signals that the parse has failed at the current instruction.
func (p *parser) parseFailed() {
	p.hasErr = true
}

// invalidDate signals that the parse succeeded, but the values where invalid
// (e.g. out of range). msg describes the validation failure.
func (p *parser) invalidDate(msg string) {
	p.hasErr = true
	p.errMsg = msg
}

func (p *parser) err(layout, value, msg string) error {
	// We call strings.Clone in this function to prevent Parse from allocating
	// in the happy path. As parts of the input appear in the error message,
	// the compiler has to mark the value argument to Parse as potentially
	// escaping. Cloning them here means the input itself never escapes. This
	// means we save an allocation in the happy path, at the cost of an extra
	// allocation in the sad path.
	//
	// It would be great if we could have our cake and eat it to, but so far,
	// the compiler is not smart enough.
	v := strings.Clone(value)
	if msg == "" {
		ve := strings.Clone(p.valEl)
		le := strings.Clone(p.inst.String())
		return &ParseError{
			Layout:     layout,
			Value:      v,
			LayoutElem: le,
			ValueElem:  ve,
		}
	}
	return &ParseError{
		Layout:  layout,
		Value:   v,
		Message: msg,
	}
}

// skipByte skips the given byte, if the input starts with it.
func (p *parser) skipByte(b byte) {
	if len(p.value) > 0 && p.value[0] == b {
		p.value = p.value[1:]
	}
}

// trimByte skips a run of the given byte.
func (p *parser) trimByte(b byte) {
	for len(p.value) > 0 && p.value[0] == b {
		p.value = p.value[1:]
	}
}

// accept a literal string, treating runs of space characters as equivalent.
func (p *parser) accept(lit string) {
	for len(lit) > 0 {
		if lit[0] == ' ' {
			if p.value != "" && p.value[0] != ' ' {
				p.parseFailed()
				return
			}
			p.trimByte(' ')
			lit = strings.TrimLeft(lit, " ")
			continue
		}
		if p.value == "" || p.value[0] != lit[0] {
			p.parseFailed()
			return
		}
		lit, p.value = lit[1:], p.value[1:]
	}
}

// atoi accepts the next i bytes of input as an integer.
func (p *parser) atoi(i int) int {
	if len(p.value) < i {
		p.parseFailed()
		return 0
	}
	v, err := strconv.Atoi(p.value[:i])
	if err != nil {
		p.parseFailed()
		return 0
	}
	p.value = p.value[i:]
	return v
}

// getnumN parses s[0:1], …, or s[0:N] (fixed forces s[0:N])
// as a decimal integer.
func (p *parser) getnumN(N int, fixed bool) int {
	var n, i int
	for i = 0; i < N && isDigit(p.value, i); i++ {
		n = n*10 + int(p.value[i]-'0')
	}
	if i == 0 || (fixed && i != N) {
		p.parseFailed()
		return 0
	}
	p.value = p.value[i:]
	return n
}

// num parses s[:1] or s[:2] (fixed forces s[:2]) as a decimal integer.
func (p *parser) num(fixed bool) int {
	return p.getnumN(2, fixed)
}

// num parser s[:1], s[:2] or s[:3] (fixed forces s[:3]) as a decimal integer.
func (p *parser) num3(fixed bool) int {
	return p.getnumN(3, fixed)
}

// peekDigit ensures that the current value starts with a digit, without
// advancing the input.
func (p *parser) peekDigit() {
	if !isDigit(p.value, 0) {
		p.parseFailed()
	}
}

// lookup a value from a table and accept a case-insensitive match.
func (p *parser) lookup(table []string) int {
	for i, v := range table {
		if len(p.value) >= len(v) && match(p.value[0:len(v)], v) {
			p.value = p.value[len(v):]
			return i
		}
	}
	p.parseFailed()
	return 0
}

// ParseError describes a problem parsing a date string.
type ParseError struct {
	Layout     string
	Value      string
	LayoutElem string
	ValueElem  string
	Message    string
}

// Error returns the string representation of a ParseError.
func (e *ParseError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("parsing date %q as %q: cannot parse %q as %q", e.Value, e.Layout, e.ValueElem, e.LayoutElem)
	}
	return fmt.Sprintf("parsing date %q: %s", e.Value, e.Message)
}
