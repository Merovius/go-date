// Copyright 2024 Axel Wagner.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package date

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

var tcs = []struct {
	year  int
	month time.Month
	day   int
	want  Date
}{
	{1, 1, 1, 0},
	{2, 1, 1, 365},
	{3, 1, 1, 730},
	{4, 1, 1, 1095},
	{5, 1, 1, 1461},

	{1, 3, 1, 59},
	{2, 3, 1, 424},
	{3, 3, 1, 789},
	{4, 3, 1, 1155},
	{5, 3, 1, 1520},

	{1, 1, 31, 30},
	{1, 2, 1, 31},
	{1, 1, 32, 31},
	{1, 1, 0, -1},
	{0, 12, 31, -1},
	{1957, 96, 104, 717408},
	{1964, 12, 104, 717408},
	{2023, 7, 14, 738714},
}

func TestOf(t *testing.T) {
	for i, tc := range tcs {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if got := Of(tc.year, tc.month, tc.day); got != tc.want {
				t.Errorf("Of(%d, %d, %d) = %d, want %d", tc.year, tc.month, tc.day, got, tc.want)
			}
			check(t, tc.year, int(tc.month), tc.day)
		})
	}
}

func TestToday(t *testing.T) {
	if got, want := Today(time.UTC), Of(time.Now().UTC().Date()); got != want {
		t.Errorf("Today(time.UTC) = %v, want %v", got, want)
	}
	if got, want := Today(time.Local), Of(time.Now().Date()); got != want {
		t.Errorf("Today(time.Local) = %v, want %v", got, want)
	}
}

func addAll(f *testing.F) {
	for _, tc := range tcs {
		f.Add(tc.year, int(tc.month), tc.day)
	}
}

func FuzzOf(f *testing.F) {
	addAll(f)
	f.Fuzz(check)
}

func FuzzMarshalText(f *testing.F) {
	addAll(f)
	f.Fuzz(func(t *testing.T, year, month, day int) {
		want := Of(year, time.Month(month), day)
		b, _ := want.MarshalText()
		t.Logf("Of(%d, %d, %d).MarshalText() = %q", year, month, day, string(b))
		var got Date
		if err := got.UnmarshalText(b); err != nil {
			t.Errorf("UnmarshalText(%q) = _, %v, want <nil>", string(b), err)
		}
		if got != want {
			t.Errorf("UnmarshalText(%q) = %v, want %v", string(b), got, want)
		}
	})
}

func FuzzUnmarshalText(f *testing.F) {
	rnd := rand.New(rand.NewSource(0))
	for i := 0; i < 100; i++ {
		b, err := Date(rnd.Intn(1e6)).MarshalText()
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		var d Date
		// we only check that UnmarshalText does not panic.
		d.UnmarshalText(b)
	})
}

func FuzzMarshalBinary(f *testing.F) {
	addAll(f)
	f.Fuzz(func(t *testing.T, year, month, day int) {
		want := Of(year, time.Month(month), day)
		b, _ := want.MarshalBinary()
		t.Logf("Of(%d, %d, %d).MarshalBinary() = %q", year, month, day, string(b))
		var got Date
		if err := got.UnmarshalBinary(b); err != nil {
			t.Errorf("UnmarshalBinary(%q) = _, %v, want <nil>", string(b), err)
		}
		if got != want {
			t.Errorf("UnmarshalBinary(%q) = %v, want %v", string(b), got, want)
		}
	})
}

func FuzzUnmarshalBinary(f *testing.F) {
	rnd := rand.New(rand.NewSource(0))
	for i := 0; i < 100; i++ {
		b, err := Date(rnd.Intn(1e6)).MarshalText()
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		var d Date
		// we only check that UnmarshalBinary does not panic.
		d.UnmarshalBinary(b)
	})
}

// check that the given year, month and day values produce the same date calculations as time.Time.
func check(t *testing.T, year, month, day int) {
	d := Of(year, time.Month(month), day)
	got := time.Date(1, 1, 1, 6, 0, 0, 0, time.UTC).AddDate(0, 0, int(d))
	want := time.Date(year, time.Month(month), day, 6, 0, 0, 0, time.UTC)
	if got != want {
		t.Errorf("Of(%d, %d, %d): %v != %v", year, month, day, got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	Y, M, D := d.Date()
	if wantY, wantM, wantD := want.Date(); Y != wantY || M != wantM || D != wantD {
		t.Errorf("Of(%d, %d, %d).Date() = %d, %d, %d, want %d, %d, %d", year, month, day, Y, M, D, wantY, wantM, wantD)
	}
	t.Logf("Of(%d, %d, %d).Date() = %d, %d, %d", year, month, day, Y, M, D)
	if d2 := Of(Y, M, D); d2 != d {
		t.Errorf("Of(%d, %d, %d) = %d, want %d", Y, M, D, d2, d)
	}
	if gotY, wantY := d.Year(), want.Year(); gotY != wantY {
		t.Errorf("Of(%d, %d, %d).Year() = %d, want %d", year, month, day, gotY, wantY)
	}
	if gotM, wantM := d.Month(), want.Month(); gotM != wantM {
		t.Errorf("Of(%d, %d, %d).Month() = %d, want %d", year, month, day, gotM, wantM)
	}
	if gotD, wantD := d.Day(), want.Day(); gotD != wantD {
		t.Errorf("Of(%d, %d, %d).Day() = %d, want %d", year, month, day, gotD, wantD)
	}
	if gotYD, wantYD := d.YearDay(), want.YearDay(); gotYD != wantYD {
		t.Errorf("Of(%d, %d, %d).YearDay() = %d, want %d", year, month, day, gotYD, wantYD)
	}
	if gotWD, wantWD := d.Weekday(), want.Weekday(); gotWD != wantWD {
		t.Errorf("Of(%d, %d, %d).Weekday() = %v, want %v", year, month, day, gotWD, wantWD)
	}
	gotIY, gotIW := d.ISOWeek()
	wantIY, wantIW := want.ISOWeek()
	if gotIY != wantIY || gotIW != wantIW {
		t.Errorf("Of(%d, %d, %d).ISOWeek() = (%d, %d), want (%d, %d)", year, month, day, gotIY, gotIW, wantIY, wantIW)
	}
}
