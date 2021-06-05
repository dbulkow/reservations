/* Copyright (c) 2021 David Bulkow */

package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	const DefaultNow = "2017-04-01 23:47:00.000000000 -0400 EDT"

	tests := []struct {
		name  string
		args  string
		now   string
		time  string
		error string
	}{
		{
			name: "increment",
			args: "+1day",
			time: "2017-04-03 00:00:00 -0400 EDT",
		},
		{
			name: "original timespec",
			args: "2",
			now:  "2017-04-01 08:00:00 -0400 EDT",
			time: "2017-04-01 10:00:00 -0400 EDT",
		},
		{
			name: "alt increment",
			args: "plus 5 days",
			time: "2017-04-07 00:00:00 -0400 EDT",
		},
		{
			name: "now increment",
			args: "now + 1 hour",
			time: "2017-04-02 01:00:00 -0400 EDT",
		},
		{
			name: "increment default to hour increment",
			args: "6",
			now:  "2017-04-01 12:00:00 -0400 EDT",
			time: "2017-04-01 18:00:00 -0400 EDT",
		},
		{
			name: "now alt caps",
			args: "NoW +1hour",
			time: "2017-04-02 01:00:00 -0400 EDT",
		},
		{
			name: "time value",
			args: "15:00",
			time: "2017-04-01 15:00:00 -0400 EDT",
		},
		{
			name: "short time value",
			args: "4pm",
			time: "2017-04-01 16:00:00 -0400 EDT",
		},
		{
			name: "pm time value",
			args: "4:30pm",
			time: "2017-04-01 16:30:00 -0400 EDT",
		},
		{
			name: "pm time value full",
			args: "04:30pm",
			time: "2017-04-01 16:30:00 -0400 EDT",
		},
		{
			name: "noon tomorrow",
			args: "noon tomorrow",
			now:  "2017-04-01 08:37:00 -0400 EDT",
			time: "2017-04-02 12:00:00 -0400 EDT",
		},
		{
			name: "day",
			args: "friday",
			time: "2017-04-07 23:47:00 -0400 EDT",
		},
		{
			name: "day time",
			args: "friday 11:30am",
			time: "2017-04-07 11:30:00 -0400 EDT",
		},
		{
			name: "day time pm",
			args: "friday 11:30pm",
			time: "2017-04-07 23:30:00 -0400 EDT",
		},
		{
			name: "date",
			args: "2019-02-22",
			time: "2019-02-22 23:47:00 -0500 EST",
		},
		{
			name: "date time",
			args: "2019-02-22 7:45pm",
			time: "2019-02-22 19:45:00 -0500 EST",
		},
		{
			name: "month day time",
			args: "april 1 11:59",
			time: "2017-04-01 11:59:00 -0400 EDT",
		},
		{
			name: "month day ordinal time",
			args: "september 2nd 11:59pm",
			time: "2017-09-02 23:59:00 -0400 EDT",
		},
		{
			name: "month day ordinal time year",
			args: "july 4rd 11:59 2018",
			time: "2018-07-04 11:59:00 -0400 EDT",
		},
		{
			name: "extra junk after timespec",
			args: "july 4rd 11:59 2018 this ia a test",
			time: "2018-07-04 11:59:00 -0400 EDT",
		},
		{
			name:  "illegal number timespec",
			args:  "15pm",
			error: "time out of range: 15:00PM",
		},
		{
			name:  "unknown token",
			args:  "whatsit",
			error: `unknown date/time value: "whatsit" (text)`,
		},
		{
			name: "24 hour time rollover",
			args: "2017-04-01 24:00",
			time: "2017-04-02 00:00:00 -0400 EDT",
		},
		{
			name: "midnight 45",
			args: "2017-04-02 00:45",
			time: "2017-04-02 00:45:00 -0400 EDT",
		},
		{
			name: "midnight 45 tak 2",
			args: "00:45",
			now:  "2017-04-02 00:00:00 -0400 EDT",
			time: "2017-04-02 00:45:00 -0400 EDT",
		},
		{
			name:  "month missing time",
			args:  "september 5 1970",
			error: `expected time, got "1970"`,
		},
		{
			name:  "month text time",
			args:  "october 31 time 1973",
			error: `expected time, got "time"`,
		},
		{
			name:  "unknown month",
			args:  "febrewairy 5rd 1999",
			error: `unknown date/time value: "febrewairy" (text)`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.now == "" {
				tc.now = DefaultNow
			}

			now, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", tc.now)
			if err != nil {
				t.Fatalf("time parse: %v", err)
			}

			tokens, err := tokenize(strings.Split(tc.args, " "))
			if err != nil {
				t.Fatalf("tokenize: %v", err)
			}

			tval, err := parseTimeSpec(now, now, tokens)
			if err != nil {
				if perr, ok := err.(*ParseError); ok {
					tokens, _ := tokenize(strings.Split(tc.args, " "))
					for i, t := range tokens.tokens {
						if perr.token.count == i+1 {
							fmt.Printf("[%s] ", t.Val)
						} else {
							fmt.Printf("%s ", t.Val)
						}
					}
					fmt.Println()
				}
				if tc.error != err.Error() {
					t.Fatalf("Error exp \"%s\" got \"%s\"\n", tc.error, err.Error())
				}
				return
			}

			timestr := tval.String()
			if tc.time != "" && tc.time != timestr {
				t.Fatalf("Time exp \"%s\" got \"%s\"\n", tc.time, timestr)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	const DefaultNow = "2017-04-01 23:47:00.000000000 -0400 EDT"

	tests := []struct {
		name  string
		args  string
		now   string
		start string
		end   string
		error string
	}{
		{
			name:  "original timespec",
			args:  "6",
			now:   "2017-04-01 07:58:00 -0400 EDT",
			start: "2017-04-01 07:58:00 -0400 EDT",
			end:   "2017-04-01 14:00:00 -0400 EDT",
		},
		{
			name:  "time plus duration",
			args:  "23:58 + 1 hour",
			start: "2017-04-01 23:58:00 -0400 EDT",
			end:   "2017-04-02 01:00:00 -0400 EDT",
		},
		{
			name:  "noon plus duration",
			args:  "noon + 5 hours",
			now:   "2017-04-01 08:00:00 -0400 EDT",
			start: "2017-04-01 12:00:00 -0400 EDT",
			end:   "2017-04-01 17:00:00 -0400 EDT",
		},
		{
			name:  "noon tomorrow plus duration",
			args:  "noon tomorrow + 5 hours",
			now:   "2017-04-01 08:00:00 -0400 EDT",
			start: "2017-04-02 12:00:00 -0400 EDT",
			end:   "2017-04-02 17:00:00 -0400 EDT",
		},
		{
			name:  "from noon tomorrow plus duration",
			args:  "from noon tomorrow + 5 hours",
			now:   "2017-04-01 08:00:00 -0400 EDT",
			start: "2017-04-02 12:00:00 -0400 EDT",
			end:   "2017-04-02 17:00:00 -0400 EDT",
		},
		{
			name:  "noon tomorrow to 5pm tomorrow",
			args:  "noon tomorrow to 5pm tomorrow",
			now:   "2017-04-01 08:00:00 -0400 EDT",
			start: "2017-04-02 12:00:00 -0400 EDT",
			end:   "2017-04-02 17:00:00 -0400 EDT",
		},
		{
			name:  "start and end 1",
			args:  "from5:45PM to noon tomorrow",
			now:   "2017-04-01 13:30:00 -0400 EDT",
			start: "2017-04-01 17:45:00 -0400 EDT",
			end:   "2017-04-02 12:00:00 -0400 EDT",
		},
		{
			name:  "start time 24",
			args:  "from 2017-04-01 24:00 to 00:45",
			start: "2017-04-02 00:00:00 -0400 EDT",
			end:   "2017-04-02 00:45:00 -0400 EDT",
		},
		{
			name:  "end time relative to start time",
			args:  "from 24:00 to 00:45",
			start: "2017-04-02 00:00:00 -0400 EDT",
			end:   "2017-04-02 00:45:00 -0400 EDT",
		},
		{
			name:  "noon tomorrow to friday 17:00",
			args:  "noon tomorrow to friday 17:00",
			now:   "2017-04-05 13:13:00 -0400 EDT",
			start: "2017-04-06 12:00:00 -0400 EDT",
			end:   "2017-04-07 17:00:00 -0400 EDT",
		},
		{
			name:  "noon tomorrow to friday 5pm",
			args:  "noon tomorrow to friday 5pm",
			now:   "2017-04-05 13:13:00 -0400 EDT",
			start: "2017-04-06 12:00:00 -0400 EDT",
			end:   "2017-04-07 17:00:00 -0400 EDT",
		},
		{
			name:  "tomorrow noon to friday 5pm",
			args:  "tomorrow noon to friday 5pm",
			now:   "2017-04-05 13:13:00 -0400 EDT",
			start: "2017-04-06 12:00:00 -0400 EDT",
			end:   "2017-04-07 17:00:00 -0400 EDT",
		},
		{
			name:  "date without time",
			args:  "2017-04-02 to friday 5pm",
			start: "2017-04-02 23:47:00 -0400 EDT",
			end:   "2017-04-07 17:00:00 -0400 EDT",
		},
		{
			name:  "month without year",
			args:  "september 5th 11:59 until 3pm",
			start: "2017-09-05 11:59:00 -0400 EDT",
			end:   "2017-09-05 15:00:00 -0400 EDT",
		},
		{
			name:  "from tomorrow 8am until 3pm",
			args:  "from tomorrow 8am until 3pm",
			start: "2017-04-02 08:00:00 -0400 EDT",
			end:   "2017-04-02 15:00:00 -0400 EDT",
		},
		{
			name:  "from 2017-04-06 8am until 3pm",
			args:  "from 2017-04-06 8am until 3pm",
			start: "2017-04-06 08:00:00 -0400 EDT",
			end:   "2017-04-06 15:00:00 -0400 EDT",
		},
		{
			name:  "from tomorrow 8:00am until tomorrow 3pm",
			args:  "from tomorrow 8:00am until 3pm tomorrow",
			start: "2017-04-02 08:00:00 -0400 EDT",
			end:   "2017-04-02 15:00:00 -0400 EDT",
		},
		{
			name:  "from 6am until 3pm",
			args:  "from 6am until 3pm",
			error: "start is in the past",
		},
		{
			name:  "4-6-2017 8am until 3pm",
			args:  "4-6-2017 8am until 3pm",
			error: "invalid date format [4-6-2017]",
		},
		{
			name:  "verify token count in error output",
			args:  "april 5st 19:00 to febrewairy 19st 7pm",
			error: `unknown date/time value: "febrewairy" (text)`,
		},
		{
			name: "for duration",
			args: "for 15 hours",
			now:  "2017-04-01 09:36:00 -0400 EDT",
			end:  "2017-04-02 01:00:00 -0400 EDT",
		},
		{
			name:  "too much input",
			args:  "from 6am tomorrow until 3pm because i want it",
			error: "extra arguments beyond timespec",
		},
		{
			name:  "October 15th 9am",
			args:  "October 15th 9am",
			error: `expected time, got "9"`,
		},
		{
			name:  "May 2nd 9:00 for 7 hours",
			now:   "2017-04-29 09:36:00 -0400 EDT",
			args:  "May 2nd 9:00 for 7 hours",
			start: "2017-05-02 09:00:00 -0400 EDT",
			end:   "2017-05-02 16:00:00 -0400 EDT",
		},
		{
			name:  "Dec 15th on Jan 29th",
			now:   "2017-01-29 09:36:00 -0500 EST",
			args:  "Dec 15th 7:00pm for 7 hours",
			start: "2017-12-15 19:00:00 -0500 EST",
			end:   "2017-12-16 02:00:00 -0500 EST",
		},
		{
			name:  "January 5th on Dec 25th 2016",
			now:   "2016-12-25 09:36:00 -0500 EST",
			args:  "January 5th 7:00pm for 7 hours",
			start: "2017-01-05 19:00:00 -0500 EST",
			end:   "2017-01-06 02:00:00 -0500 EST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.now == "" {
				tc.now = DefaultNow
			}

			now, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", tc.now)
			if err != nil {
				t.Fatalf("time parse: %v", err)
			}

			start, end, err := ParseRange(now, strings.Split(tc.args, " "))
			if err != nil {
				if perr, ok := err.(*ParseError); ok {
					tokens, _ := tokenize(strings.Split(tc.args, " "))
					for i, t := range tokens.tokens {
						if perr.token.count == i+1 {
							fmt.Printf("[%s] ", t.Val)
						} else {
							fmt.Printf("%s ", t.Val)
						}
					}
					fmt.Println()
				}
				if tc.error != err.Error() {
					t.Fatalf("Error exp \"%s\" got \"%s\"\n", tc.error, err.Error())
				}
			}

			// fmt.Println(now.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
			// fmt.Println(start.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))

			startstr := start.String()
			if tc.start != "" && tc.start != startstr {
				t.Fatalf("Start exp \"%s\" got \"%s\"\n", tc.start, startstr)
			}

			endstr := end.String()
			if tc.end != "" && tc.end != endstr {
				t.Fatalf("End exp \"%s\" got \"%s\"\n", tc.end, endstr)
			}
		})
	}
}

func TestLeapYear(t *testing.T) {
	years := []struct {
		year int
		leap bool
	}{
		{year: 2016, leap: true},
		{year: 1900, leap: false},
		{year: 1700, leap: false},
		{year: 2000, leap: true},
		{year: 2015, leap: false},
	}

	for _, y := range years {
		if leap := isLeapYear(y.year); leap != y.leap {
			t.Errorf("Year %d got %t exp %t", y.year, leap, y.leap)
		}
	}
}

func TestDateValid(t *testing.T) {
	tests := []struct {
		year  int
		month int
		day   int
		valid bool
	}{
		{year: 2016, month: 2, day: 28, valid: true},
		{year: 2016, month: 2, day: 29, valid: true},
		{year: 2016, month: 2, day: 30, valid: false},
		{year: 1700, month: 2, day: 28, valid: true},
		{year: 1700, month: 2, day: 29, valid: false},
		{year: 1700, month: 2, day: 30, valid: false},
		{year: 2000, month: 2, day: 28, valid: true},
		{year: 2000, month: 2, day: 29, valid: true},
		{year: 2000, month: 2, day: 30, valid: false},
		{year: 2015, month: 2, day: 28, valid: true},
		{year: 2015, month: 2, day: 29, valid: false},
		{year: 2015, month: 2, day: 30, valid: false},
		{year: 2015, month: 1, day: 31, valid: true},
		{year: 2015, month: 1, day: 32, valid: false},
		{year: 2015, month: 4, day: 30, valid: true},
		{year: 2015, month: 4, day: 31, valid: false},
		{year: 2015, month: 0, day: 1, valid: false},
		{year: 2015, month: 14, day: 1, valid: false},
		{year: 2015, month: 4, day: 0, valid: false},
	}

	for _, td := range tests {
		tok := &token{Year: td.year, Month: td.month, Day: td.day}
		err := tok.dateValid()
		if (err == nil) != td.valid {
			t.Errorf("Date %d-%d-%d got %t exp %t %v", td.year, td.month, td.day, err == nil, td.valid, err)
		}
	}
}
