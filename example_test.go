// Copyright 2024 Axel Wagner.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package date_test

import (
	"fmt"
	"time"

	"gonih.org/date"
)

// ExampleOf demonstrates some useful patterns when using Of.
func ExampleOf() {
	// Create a fixed date:
	d := date.Of(2023, 12, 31)
	fmt.Println(d)

	// Dates are noramlized:
	d = date.Of(2023, 12, 40)
	fmt.Println(d)

	// Get the Date of a time.Time:
	t := time.Date(2024, 1, 10, 13, 24, 42, 0, time.UTC)
	d = date.Of(t.Date())
	fmt.Println(d)

	// Get the Date from a unix timestamp.
	// Note that time.Unix returns a local time, for reproducibility, we
	// convert it to UTC:
	d = date.Of(time.Unix(1672528154, 0).UTC().Date())
	fmt.Println(d)

	// Output:
	// 2023-12-31
	// 2024-01-09
	// 2024-01-10
	// 2022-12-31
}

// ExampleDiffDates demonstrates how to check if two dates differ by a given
// amount.
func ExampleDiffDates() {
	// When comparing by number of days, we can just check their difference:
	if d1, d2 := date.Of(2024, 3, 5), date.Of(2024, 2, 5); d2-d1 < 31 {
		fmt.Printf("%v and %v are less than 31 days apart.\n", d1, d2)
	}

	// However, if we want to check if they are a month apart, we have to be careful:
	if d1, d2 := date.Of(2024, 3, 5), date.Of(2024, 2, 5); d2-d1 >= 30 {
		// Does not print.
		fmt.Printf("%v and %v are at least 30 days apart.\n", d1, d2)
	}
	// Instead, we use AddDate:
	if d1, d2 := date.Of(2024, 3, 5), date.Of(2024, 2, 5); d1.AddDate(0, 1, 0) >= d2 {
		fmt.Printf("%v and %v are at least a month apart.\n", d1, d2)
	}

	// Similarly, we need to be careful when comparing years:
	if d1, d2 := date.Of(2024, 2, 5), date.Of(2025, 2, 5); d2-d1 <= 365 {
		// Does not print.
		fmt.Printf("%v and %v are at most 365 days apart.\n", d1, d2)
	}
	// Instead, we again use AddDate:
	if d1, d2 := date.Of(2024, 2, 5), date.Of(2025, 2, 5); d1.AddDate(1, 0, 0) <= d2 {
		fmt.Printf("%v and %v are at most a year apart.\n", d1, d2)
	}

	// Output:
	// 2024-03-05 and 2024-02-05 are less than 31 days apart.
	// 2024-03-05 and 2024-02-05 are at least a month apart.
	// 2024-02-05 and 2025-02-05 are at most a year apart.
}

// ExampleParse demonstrates the usage of Parse.
func ExampleParse() {
	// Parse date according to RFC3339.
	fmt.Println(date.Parse(date.RFC3339, "2024-05-14"))

	// Parse the same date in E-Mail format.
	fmt.Println(date.Parse(date.RFC1123, "14 May 2024"))

	// Parse the same date in US date format
	fmt.Println(date.Parse("01/02/06", "05/14/24"))

	// Unlike Of, which normalizes, Parse validates ranges.
	fmt.Println(date.Parse(date.RFC3339, "2024-13-01"))
	fmt.Println(date.Parse(date.RFC3339, "2024-02-29"))
	fmt.Println(date.Parse(date.RFC3339, "2023-02-29"))

	// But it does not validate whether the specified day of the week is
	// correct for the specified date, for compatibility with time.Time.
	d, err := date.Parse("Monday 2006-01-02", "Friday 2024-02-25")
	fmt.Println(d, err, d.Weekday())

	// Output:
	// 2024-05-14 <nil>
	// 2024-05-14 <nil>
	// 2024-05-14 <nil>
	// 0001-01-01 parsing date "2024-13-01": month out of range
	// 2024-02-29 <nil>
	// 0001-01-01 parsing date "2023-02-29": day out of range
	// 2024-02-25 <nil> Sunday
}
