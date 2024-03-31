// Copyright 2009 The Go Authors.
// Copyright 2024 Axel Wagner.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package date contains a Gregorian date type.
//
// When handling calendar dates, the standard library time package has some
// shortcomings:
//
//   - A time.Time is always a specific point in time in a specific timezone.
//     It is not canonically clear, what the right clock time is, when using them
//     to represent a date. It's possible to just choose a fixed time (e.g.
//     6am) and timezone (e.g. UTC) but that requires some syntactic overhead
//     to always specify them.
//   - There is no easy way to get the number of days between two time.Time values.
//     time.Time.Sub gives you a time.Duration, which can only represent ~292 years.
//     Even then, it feels icky to divide a time.Duration into 24h time spans,
//     given all the surprising things we do with daylight savings time and
//     leap seconds (it *should* be fine, if sticking with UTC, but I'm
//     not completely sure, which is the point).
//   - A time.Time doesn't format nice and doesn't Marshal/Unmarshal into nice
//     text, as the clock part is always serialized as well.
//   - There is a small overhead of carrying around the clock data, which is
//     never used, but still computed with.
//
// This package provides a simple Date type that is intended to be compatible
// with the time package, but ignores the clock and timezone aspects. The date
// calculation code is largely copied, so it should make the same assumptions
// and have similar edge-cases.
//
// There is no equivalent to time.Duration. The correct unit for that would be
// a Day. Given that Date already represents a number of days, it can be
// directly compared/added to/subtracted from.
package date

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

// Computations on time are essentially copied verbatim from the standard
// library. See this comment for explanations:
// https://cs.opensource.google/go/go/+/refs/tags/go1.20.6:src/time/time.go;l=353
// Some calculations are simplified by the fact that we don't care about clock
// times and timezones.

const (
	// The unsigned zero year for internal calculations.
	// Must be 1 mod 400, and times before it will not compute correctly, but
	// otherwise can be changed at will.
	absoluteZeroYear = -292277022399

	// The year of the zero Date.
	internalYear = 1

	// Offsets to convert between internal or absolute times.
	absoluteToInternal = (absoluteZeroYear - internalYear) * 365.2425
	internalToAbsolute = -absoluteToInternal

	// Days in a given period of years.
	daysPer400Years = 146097
	daysPer100Years = 36524
	daysPer4Years   = 1461
)

// daysBefore[m] counts the number of days in a non-leap year before month m
// begins. There is an entry for m=12, conuting the number of days before
// January of next year (365).
var daysBefore = [...]int{
	0,
	31,
	31 + 28,
	31 + 28 + 31,
	31 + 28 + 31 + 30,
	31 + 28 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30 + 31,
}

// daysInMonth counts the (maximum) numbers of days in a given month.
var daysInMonth = [...]int{
	time.January:   31,
	time.February:  29,
	time.March:     31,
	time.April:     30,
	time.May:       31,
	time.June:      30,
	time.July:      31,
	time.August:    31,
	time.September: 30,
	time.October:   31,
	time.November:  30,
	time.December:  31,
}

func daysIn(m time.Month, year int) int {
	if m == time.February && isLeap(year) {
		return 29
	}
	return int(daysBefore[m] - daysBefore[m-1])
}

// absDate computes the year, day of year and when full=true, the month and day
// in which an absolute date occurs.
func absDate(abs uint64, full bool) (year int, month time.Month, day int, yday int) {
	d := abs

	// Account for 400 year cycles.
	n := d / daysPer400Years
	y := 400 * n
	d -= daysPer400Years * n

	// Cut off 100-year cycles.
	// The last cycle has one extra leap year, so on the last day
	// of that year, day / daysPer100YearsYears will be 4 instead of 3.
	// Cut it back down to 3 by subtracting n>>2.
	n = d / daysPer100Years
	n -= n >> 2
	y += 100 * n
	d -= daysPer100Years * n

	// Cut off 4-year cycles.
	// The last cycle has a missing leap year, which does not
	// affect the computation.
	n = d / daysPer4Years
	y += 4 * n
	d -= daysPer4Years * n

	// Cut off years within a 4-year cycle.
	// The last year is a leap year, so on the last day of that year,
	// day / 365 will be 4 instead of 3. Cut it back down to 3
	// by subtracting n>>2.
	n = d / 365
	n -= n >> 2
	y += n
	d -= 365 * n

	year = int(int64(y) + absoluteZeroYear)
	yday = int(d)

	if !full {
		return
	}

	day = yday
	if isLeap(year) {
		// Leap year
		switch {
		case day > 31+29-1:
			// After leap day; pretend it wasn't there.
			day--
		case day == 31+29-1:
			// Leap day.
			month = time.February
			day = 29
			return
		}
	}

	// Estimate month on assumption that every month has 31 days.
	// The estimate may be too low by at most one month, so adjust.
	month = time.Month(day / 31)
	end := int(daysBefore[month+1])
	var begin int
	if day >= end {
		month++
		begin = end
	} else {
		begin = int(daysBefore[month])
	}

	month++ // because January is 1
	day = day - begin + 1
	return year, month, day, yday
}

// daysSinceEpoch takes a year and returns the number of days from the absolute
// epoch to the start of that year. This is basically (year - zeroYear) * 365,
// but accounting for leap days.
func daysSinceEpoch(year int) int {
	y := year - absoluteZeroYear

	n := y / 400
	y -= 400 * n
	d := daysPer400Years * n

	n = y / 100
	y -= 100 * n
	d += daysPer100Years * n

	n = y / 4
	y -= 4 * n
	d += daysPer4Years * n

	n = y
	d += 365 * n

	return int(d)
}

func isLeap(year int) bool {
	return (year%4 == 0 && (year%100 != 0 || year%400 == 0))
}

// norm returns nhi, nlo such that
//
//	hi * base + lo == nhi * base + nlo
//	0 <= nlo < base
func norm(hi, lo, base int) (nhi, nlo int) {
	if lo < 0 {
		n := (-lo-1)/base + 1
		hi -= n
		lo += n * base
	}
	if lo >= base {
		n := lo / base
		hi += n
		lo -= n * base
	}
	return hi, lo
}

// A Date represents a date, as the number of days since 0001-01-01. The zero
// value of Date is thus the same date as the zero value of time.Time. The
// Gregorian calendar is used, even for dates lying before its introduction.
//
// Dates can be compared using Go's arithmetic operators.
type Date int

// Of returns the Date correspomding to the given date.
//
// The arguments may be outside their usual ranges and will be normalized
// during the conversion, just as for [time.Date]. For example, October 32
// converts to November 1.
func Of(year int, month time.Month, day int) Date {
	m := int(month) - 1
	year, m = norm(year, m, 12)
	month = time.Month(m) + 1

	d := daysSinceEpoch(year)
	d += daysBefore[month-1]
	if isLeap(year) && month >= time.March {
		d++
	}

	d += day - 1

	return Date(d - internalToAbsolute)
}

// Today returns the current date in the given location.
func Today(loc *time.Location) Date {
	return Of(time.Now().In(loc).Date())
}

// abs returns the absolute date of d.
func (d Date) abs() uint64 {
	return uint64(d + internalToAbsolute)
}

// AddDate returns the time corresponding to adding the given number of years,
// months, and days to d. For example, AddDate(-1, 2, 3) applied to January 1,
// 2011 returns March 4, 2010.
//
// AddDate normalizes its result in the same way that Date does, so, for
// example, adding one month to October 31 yields December 1, the normalized
// form for November 31.
//
// AdDate(0, 0, days) is equivalent to d+Date(days).
func (d Date) AddDate(years, months, days int) Date {
	year, month, day := d.Date()
	return Of(year+years, month+time.Month(months), day+days)
}

// Date returns the normalized year, month and day specified by d.
func (d Date) Date() (year int, month time.Month, day int) {
	year, month, day, _ = absDate(d.abs(), true)
	return year, month, day
}

// Day returns the day of the month of d.
func (d Date) Day() int {
	_, _, day := d.Date()
	return day
}

// GoString implements fmt.GoStringer and formats d to be printed in Go source code.
func (d Date) GoString() string {
	year, month, day := d.Date()
	return fmt.Sprintf("date.Of(%d, %d, %d)", year, month, day)
}

// ISOWeek returns the ISO 8601 year and week number in which d occurs. Week
// ranges from 1 to 53. Jan 01 to Jan 03 of year n might belong to week 52 or
// 53 of year n-1, and Dec 29 to Dec 31 might belong to week 1 of year n+1.
func (d Date) ISOWeek() (year, week int) {
	// See this comment for an explanation:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.20.6:src/time/time.go;l=544

	offset := time.Thursday - d.Weekday()
	if offset == 4 {
		offset = -3
	}
	d += Date(offset)
	year, _, _, yday := absDate(d.abs(), false)
	return year, yday/7 + 1
}

// MarshalBinary implements the encoding.BinaryMarshaler interface. The date is
// represented as a [binary.Varint] representing the number of days since
// 0001-01-01.
func (d Date) MarshalBinary() ([]byte, error) {
	b := make([]byte, binary.MaxVarintLen64)
	return b[:binary.PutVarint(b, int64(d))], nil
}

// MarshalText implements the encoding.TextMarshaler interface. The date is
// formatted in ISO 8601 format.
func (d Date) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// Month returns the month of the year specified by d.
func (d Date) Month() time.Month {
	_, month, _ := d.Date()
	return month
}

// String returns the date formatted as ISO 8601.
//
// The returned string is meant for debugging; for a stable serialized
// representation, use d.MarshalText or t.MarshalBinary.
func (d Date) String() string {
	return d.Format(RFC3339)
}

// Time returns the given moment in time in the given location.
func (d Date) Time(hour, min, sec, nsec int, loc *time.Location) time.Time {
	return time.Date(1, 1, 1+int(d), hour, min, sec, nsec, loc)
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (d *Date) UnmarshalBinary(b []byte) error {
	v, i := binary.Varint(b)
	switch {
	case i == 0:
		return errors.New("encoded date truncated")
	case i < 0 || int64(int(v)) != v:
		return errors.New("encoded date overflows int")
	case i != len(b):
		return errors.New("extra data after date")
	}
	*d = Date(v)
	return nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface. The date
// must be in ISO 8601 format.
func (d *Date) UnmarshalText(b []byte) error {
	v, err := Parse(RFC3339, string(b))
	if err == nil {
		*d = v
	}
	return err
}

// Weekday returns the day of the week specified by d.
func (d Date) Weekday() time.Weekday {
	return (time.Monday + time.Weekday(d.abs())) % 7 // 0001-01-01 was a Monday
}

// Year returns the year in which d occurs.
func (d Date) Year() int {
	year, _, _, _ := absDate(d.abs(), false)
	return year
}

// YearDay returns the day of the year specified by d, in the range [1,365] for
// non-leap years, and [1,366] in leap years.
func (d Date) YearDay() int {
	_, _, _, yday := absDate(d.abs(), false)
	return yday + 1
}
