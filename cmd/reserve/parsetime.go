/* Copyright (c) 2021 David Bulkow */

package main

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

/*
Grammar:

	plus:         'plus' | '+' | 'for'
	num:          [0-9]+
	hour:         'h' | 'hour' | 'hours'
	day:          'd' | 'day' | 'days'
	week:         'w' | 'week' | 'weeks'
	rel:          hour | day | week
	duration:     number [ rel ]
	dayname:      mon | tue | wed | thu ...
	month:        jan | feb | mar | apr ...
	date:         yyyy-mm-dd
	time_mod:     am | pm
	short_time:   number [ time_mod ]
	std_time:     hh:mm [ time_mod ]
	time:         hh:mm | short_time
	ordinal:      nd | rd | st | th
	datetime:     date time
	longdate:     month num [ ordinal ] std_time [ yyyy ]
	dayspec:      [ 'next' ] dayname time
	tomorrow:     time 'tomorrow' | 'tomorrow' time
	timespec:     time | longdate | datetime | tomorrow

	plustime:     [ plus ] duration
	explicit_end: ( until | to ) timespec
	start_plus:   timespec plus duration
	start_end:    timespec ( until | to ) timespec

	now           now
	noon          12:00
	midnight      00:00
	eod           17:00
	tomorrow      time + 24 hours

Example time specifications

	+1day
	2
	plus 5 days
	now + 1 hour
	6
	NoW +1hour
	15:00
	4pm
	4:30pm
	04:30pm
	noon tomorrow
	friday
	friday 11:30am
	friday 11:30pm
	2019-02-22
	2019-02-22 7:45pm
	april 1 11:59
	september 2nd 11:59pm
	july 4rd 11:59 2018
	july 4rd 11:59 2018 this is a test
	23:58 + 1 hour
	noon + 5 hours
	noon tomorrow + 5 hours
	from noon tomorrow + 5 hours
	noon tomorrow to 5pm tomorrow
	from5:45PM to noon tomorrow

Use of 'tomorrow' is relative to _now_ rather than the start date.

End times without a date will be relative to the start time.
*/

type token struct {
	Val    string
	Num    int
	Type   int
	Year   int
	Month  int
	Day    int
	Hour   int
	Minute int
	count  int
}

const (
	TokNone = iota
	TokText
	TokNumber
	TokTime
	TokPlus
	TokNow
	TokNext
	TokFrom
	TokTo
	TokUntil
	TokFor
	TokDay
	TokMonth
	TokDate
	TokTomorrow
	TokNoon
	TokMidnight
	TokEOD
	TokAM
	TokPM
	TokRelHour
	TokRelDay
	TokRelWeek
	TokOrdinal
)

var tokTypes = map[int]string{
	TokNone:     "none",
	TokText:     "text",
	TokNumber:   "number",
	TokTime:     "time",
	TokPlus:     "plus",
	TokNow:      "now",
	TokNext:     "next",
	TokFrom:     "from",
	TokTo:       "to",
	TokUntil:    "until",
	TokFor:      "for",
	TokDay:      "day",
	TokMonth:    "month",
	TokDate:     "date",
	TokTomorrow: "tomorrow",
	TokNoon:     "noon",
	TokMidnight: "midnight",
	TokEOD:      "eod",
	TokAM:       "am",
	TokPM:       "pm",
	TokRelHour:  "hour",
	TokRelDay:   "day",
	TokRelWeek:  "week",
	TokOrdinal:  "ord",
}

var Text2Tok = map[string]int{
	"plus":      TokPlus,
	"for":       TokFor,
	"next":      TokNext,
	"now":       TokNow,
	"from":      TokFrom,
	"to":        TokTo,
	"until":     TokUntil,
	"mon":       TokDay,
	"tue":       TokDay,
	"wed":       TokDay,
	"thu":       TokDay,
	"fri":       TokDay,
	"sat":       TokDay,
	"sun":       TokDay,
	"monday":    TokDay,
	"tuesday":   TokDay,
	"wednesday": TokDay,
	"thursday":  TokDay,
	"friday":    TokDay,
	"saturday":  TokDay,
	"sunday":    TokDay,
	"jan":       TokMonth,
	"feb":       TokMonth,
	"mar":       TokMonth,
	"apr":       TokMonth,
	"may":       TokMonth,
	"jun":       TokMonth,
	"jul":       TokMonth,
	"aug":       TokMonth,
	"sep":       TokMonth,
	"oct":       TokMonth,
	"nov":       TokMonth,
	"dec":       TokMonth,
	"january":   TokMonth,
	"february":  TokMonth,
	"march":     TokMonth,
	"april":     TokMonth,
	"june":      TokMonth,
	"july":      TokMonth,
	"august":    TokMonth,
	"september": TokMonth,
	"october":   TokMonth,
	"november":  TokMonth,
	"december":  TokMonth,
	"tomorrow":  TokTomorrow,
	"noon":      TokNoon,
	"midnight":  TokMidnight,
	"eod":       TokEOD,
	"am":        TokAM,
	"pm":        TokPM,
	"h":         TokRelHour,
	"hour":      TokRelHour,
	"hours":     TokRelHour,
	"d":         TokRelDay,
	"day":       TokRelDay,
	"days":      TokRelDay,
	"w":         TokRelWeek,
	"week":      TokRelWeek,
	"weeks":     TokRelWeek,
	"nd":        TokOrdinal,
	"rd":        TokOrdinal,
	"st":        TokOrdinal,
	"th":        TokOrdinal,
}

var Days = map[string]int{
	"sunday":    0,
	"sun":       0,
	"monday":    1,
	"mon":       1,
	"tuesday":   2,
	"tue":       2,
	"wednesday": 3,
	"wed":       3,
	"thursday":  4,
	"thu":       4,
	"friday":    5,
	"fri":       5,
	"saturday":  6,
	"sat":       6,
}

var Months = map[string]int{
	"january":   1,
	"jan":       1,
	"february":  2,
	"feb":       2,
	"march":     3,
	"mar":       3,
	"april":     4,
	"apr":       4,
	"may":       5,
	"june":      6,
	"jun":       6,
	"july":      7,
	"jul":       7,
	"august":    8,
	"aug":       8,
	"september": 9,
	"sep":       9,
	"october":   10,
	"oct":       10,
	"november":  11,
	"nov":       11,
	"december":  12,
	"dec":       12,
}

// months with 31 days
var Months31 = map[int]bool{
	1:  true,
	3:  true,
	5:  true,
	7:  true,
	8:  true,
	10: true,
	12: true,
}

type ParseError struct {
	msg         string
	token       *token
	endOfInput  bool
	badToken    bool
	wrongToken  bool
	notFound    bool
	invalid     bool
	dateInvalid bool
}

func (e *ParseError) Error() string     { return e.msg }
func (e *ParseError) EndOfInput() bool  { return e.endOfInput }
func (e *ParseError) BadToken() bool    { return e.badToken }
func (e *ParseError) WrongToken() bool  { return e.wrongToken }
func (e *ParseError) NotFound() bool    { return e.notFound }
func (e *ParseError) Invalid() bool     { return e.invalid }
func (e *ParseError) DateInvalid() bool { return e.dateInvalid }

func isLeapYear(year int) bool {
	if year%4 == 0 {
		if year%100 == 0 {
			if year%400 == 0 {
				return true
			}
			return false
		}
		return true
	}
	return false
}

func (t *token) dateValid() error {
	if t.Month == 0 {
		return &ParseError{
			msg:         "month is zero",
			dateInvalid: true,
			token:       t,
		}
	}

	if t.Month > 12 {
		return &ParseError{
			msg:         "month too large",
			dateInvalid: true,
			token:       t,
		}
	}

	if t.Day == 0 {
		return &ParseError{
			msg:         "day is zero",
			dateInvalid: true,
			token:       t,
		}
	}

	if t.Month == 2 {
		if isLeapYear(t.Year) {
			if t.Day > 29 {
				return &ParseError{
					msg:         "day too large",
					dateInvalid: true,
					token:       t,
				}
			}
		} else {
			if t.Day > 28 {
				return &ParseError{
					msg:         "day too large",
					dateInvalid: true,
					token:       t,
				}
			}
		}
	}

	_, ok := Months31[t.Month]
	if (ok && t.Day > 31) || (!ok && t.Day > 30) {
		return &ParseError{
			msg:         "day too large",
			dateInvalid: true,
			token:       t,
		}
	}

	return nil
}

func (t *token) String() string {
	return fmt.Sprintf("(%d) %s", t.count, t.Val)
}

type fifo struct {
	tokens []*token
	count  int
}

func NewFifo() *fifo {
	return &fifo{tokens: make([]*token, 0)}
}

func (s *fifo) Push(tok *token) error {
	switch tok.Type {
	case TokText:
		t, ok := Text2Tok[tok.Val]
		if ok {
			tok.Type = t
			switch t {
			case TokNoon:
				tok.Type = TokTime
				tok.Hour = 12
				tok.Minute = 0
			case TokMidnight:
				tok.Type = TokTime
				tok.Hour = 0
				tok.Minute = 0
			case TokEOD:
				tok.Type = TokTime
				tok.Hour = 17
				tok.Minute = 0
			}
		}
	case TokNumber:
		tok.Num, _ = strconv.Atoi(tok.Val)
	case TokTime:
		p := strings.Split(tok.Val, ":")
		tok.Hour, _ = strconv.Atoi(p[0])
		tok.Minute, _ = strconv.Atoi(p[1])
	case TokDate:
		// yyyy-mm-dd  iso 8601
		valid := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

		p := strings.Split(tok.Val, "-")

		if !valid.MatchString(tok.Val) {
			return fmt.Errorf("invalid date format [%s]", tok.Val)
		}
		tok.Year, _ = strconv.Atoi(p[0])
		tok.Month, _ = strconv.Atoi(p[1])
		tok.Day, _ = strconv.Atoi(p[2])

		if err := tok.dateValid(); err != nil {
			return fmt.Errorf("invalid date: [%s] (%v)", tok.Val, err)
		}
	case TokNone:
		return nil
	}
	s.count++
	tok.count = s.count
	s.tokens = append(s.tokens, tok)
	return nil
}

func (s *fifo) Pop() (*token, error) {
	l := len(s.tokens)
	if l == 0 {
		return nil, &ParseError{msg: fmt.Sprintf("end of input"), endOfInput: true}
	}

	tok := s.tokens[0]
	s.tokens = s.tokens[1:]

	// fmt.Printf("%-10s %s\n", tok.Val, tokTypes[tok.Type])

	return tok, nil
}

func (s *fifo) Peek() (*token, error) {
	l := len(s.tokens)
	if l == 0 {
		return nil, &ParseError{msg: "end of input", endOfInput: true}
	}

	return s.tokens[0], nil
}

func (s *fifo) GetToken(toktype int) (*token, error) {
	tok, err := s.Peek()
	if err != nil {
		return nil, err
	}
	if tok.Type != toktype {
		return tok, &ParseError{msg: "not found", notFound: true}
	}
	s.Pop()

	return tok, nil
}

func tokenize(args []string) (*fifo, error) {
	tokens := NewFifo()

	reader := strings.NewReader(strings.ToLower(strings.Join(args, " ")))

	tok := &token{}

	for {
		r, n, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				if err := tokens.Push(tok); err != nil {
					return nil, err
				}
				break
			}
			return nil, fmt.Errorf("readrune: %v\n", err)
		}
		if n == 0 {
			if err := tokens.Push(tok); err != nil {
				return nil, err
			}
			break
		}

		switch tok.Type {
		case TokText:
			if unicode.IsLetter(r) {
				tok.Val = tok.Val + string(r)
				continue
			}
			// promote to date token
			if r == '-' {
				tok.Val = tok.Val + string(r)
				tok.Type = TokDate
				continue
			}

		case TokNumber:
			if unicode.IsDigit(r) {
				tok.Val = tok.Val + string(r)
				continue
			}
			// promote to time token
			if r == ':' {
				tok.Val = tok.Val + string(r)
				tok.Type = TokTime
				continue
			}
			// promote to date token
			if r == '-' {
				tok.Val = tok.Val + string(r)
				tok.Type = TokDate
				continue
			}

		case TokDate:
			if unicode.IsDigit(r) || r == '-' {
				tok.Val = tok.Val + string(r)
				continue
			}

			break

		case TokTime:
			if unicode.IsDigit(r) {
				tok.Val = tok.Val + string(r)
				continue
			}

			break
		}

		if err := tokens.Push(tok); err != nil {
			return nil, err
		}

		switch {
		case unicode.IsLetter(r):
			tok = &token{Val: string(r), Type: TokText}
			continue
		case unicode.IsDigit(r):
			tok = &token{Val: string(r), Type: TokNumber}
			continue
		case r == '+':
			tok = &token{Val: string(r), Type: TokPlus}
			continue
		case r == ' ':
			tok = &token{}
			continue
		}

		return nil, fmt.Errorf("malformed value: type %s, val \"%c\"", tokTypes[tok.Type], r)
	}

	return tokens, nil
}

func isRelative(tok *token) bool {
	switch tok.Type {
	case TokRelHour:
		return true
	case TokRelDay:
		return true
	case TokRelWeek:
		return true
	}
	return false
}

func parseRelativeDuration(tokens *fifo) (time.Duration, error) {
	var d time.Duration

	num, err := tokens.GetToken(TokNumber)
	if err != nil {
		if perr, ok := err.(*ParseError); ok && perr.NotFound() {
			return 0, &ParseError{
				msg:     "expect numeric value in duration",
				invalid: true,
				token:   num,
			}
		}
		if perr, ok := err.(*ParseError); ok && perr.EndOfInput() {
			return 0, &ParseError{
				msg:        "expect duration",
				endOfInput: true,
			}
		}
	}

	rel, err := tokens.Peek()
	if err != nil {
		if perr, ok := err.(*ParseError); ok {
			if perr.EndOfInput() {
				rel = &token{Type: TokRelHour, Val: "hours"}
			} else {
				return 0, err
			}
		}
	}
	if err == nil {
		tokens.Pop()
	}
	if !isRelative(rel) {
		return 0, &ParseError{
			msg:     fmt.Sprintf("invalid duration qualifier: %s", rel.Val),
			invalid: true,
			token:   rel,
		}
	}

	dur := 0
	switch rel.Type {
	case TokRelHour:
		dur = num.Num
	case TokRelDay:
		dur = 24 * num.Num
	case TokRelWeek:
		dur = 24 * 7 * num.Num
	default:
		return 0, &ParseError{
			msg:     fmt.Sprintf("unsupported relative duration: %s", rel.Val),
			invalid: true,
			token:   rel,
		}
	}

	d, err = time.ParseDuration(fmt.Sprintf("%dh", dur))
	if err != nil {
		panic(fmt.Sprintf("ParseDuration failed: %v", err))
	}

	return d, nil
}

type Time struct {
	time time.Time
}

func NewTime(base time.Time) *Time {
	t := &Time{
		time: time.Date(
			base.Year(),
			base.Month(),
			base.Day(),
			base.Hour(),
			base.Minute(),
			int(0), int(0), time.Local,
		),
	}
	return t
}

func (t *Time) Year(year int) *Time {
	t.time = time.Date(
		year,
		t.time.Month(),
		t.time.Day(),
		t.time.Hour(),
		t.time.Minute(),
		int(0), int(0), time.Local,
	)
	return t
}

func (t *Time) Month(month int) *Time {
	t.time = time.Date(
		t.time.Year(),
		time.Month(month),
		t.time.Day(),
		t.time.Hour(),
		t.time.Minute(),
		int(0), int(0), time.Local,
	)
	return t
}

func (t *Time) Day(day int) *Time {
	t.time = time.Date(
		t.time.Year(),
		t.time.Month(),
		day,
		t.time.Hour(),
		t.time.Minute(),
		int(0), int(0), time.Local,
	)
	return t
}

func (t *Time) Hour(hour int) *Time {
	t.time = time.Date(
		t.time.Year(),
		t.time.Month(),
		t.time.Day(),
		hour,
		t.time.Minute(),
		int(0), int(0), time.Local,
	)
	return t
}

func (t *Time) Minute(minute int) *Time {
	t.time = time.Date(
		t.time.Year(),
		t.time.Month(),
		t.time.Day(),
		t.time.Hour(),
		minute,
		int(0), int(0), time.Local,
	)
	return t
}

func (t *Time) Tomorrow() *Time {
	t.time = t.time.AddDate(0, 0, 1)
	return t
}

func (t *Time) AddMonths(months int) *Time {
	t.time = t.time.AddDate(0, months, 0)
	return t
}

func (t *Time) AddDays(days int) *Time {
	t.time = t.time.AddDate(0, 0, days)
	return t
}

func (t *Time) AddMinutes(d time.Duration) *Time {
	ts := t.time.Add(d).Round(30 * time.Minute)

	if ts.Sub(t.time) < d {
		ts = t.time.Add(d)
		roundUp(&ts)
	}

	t.time = ts
	return t
}

func (t *Time) AddHours(hours int) *Time {
	d := time.Duration(hours) * time.Hour

	ts := t.time.Add(d)

	if ts.Sub(t.time) < d {
		ts = ts.Add(d)
	}

	roundUp(&ts)
	t.time = ts

	return t
}

const (
	TimeOnly      = true
	TimeAndNumber = false
)

func (t *Time) Parse(tokens *fifo, timeOnly bool) (*Time, error) {
	ts, err := tokens.Peek()
	if err != nil {
		return t, err
	}

	t.time = time.Date(
		t.time.Year(),
		t.time.Month(),
		t.time.Day(),
		t.time.Hour(),
		t.time.Minute(),
		int(0), int(0), time.Local,
	)

	if ts.Type == TokTime {
		ts, err := tokens.GetToken(TokTime)
		if err != nil {
			return t, err
		}
		t.Hour(ts.Hour).Minute(ts.Minute)
	} else if !timeOnly && ts.Type == TokNumber {
		tokens.Pop()
		t.Hour(ts.Num).Minute(0)
	} else {
		return t, &ParseError{
			msg:     fmt.Sprintf("expected time, got \"%s\"", ts.Val),
			invalid: true,
			token:   ts,
		}
	}

	if _, err := t.ParsePM(tokens); err != nil {
		return t, err
	}

	return t, nil
}

func (t *Time) ParsePM(tokens *fifo) (*Time, error) {
	if _, err := tokens.GetToken(TokAM); err == nil {
		return t, nil
	}

	tok, err := tokens.GetToken(TokPM)
	if err == nil {
		if t.time.Hour() >= 12 {
			return nil, &ParseError{
				msg:     fmt.Sprintf("time out of range: %02.2d:%02.2dPM", t.time.Hour(), t.time.Minute()),
				invalid: true,
				token:   tok,
			}
		}
		t.time = t.time.Add(time.Hour * 12)
	}

	return t, nil
}

func (t *Time) Time() time.Time {
	return t.time
}

func (t *Time) String() string {
	if t == nil {
		return "<nil>"
	}
	return t.Time().String()
}

func roundUp(val *time.Time) {
	t := *val
	*val = t.Add(14 * time.Minute).Round(30 * time.Minute)
}

func parseTimeSpec(now time.Time, start time.Time, tokens *fifo) (*Time, error) {
	var timespec *Time

loop:
	for {
		t, err := tokens.Pop()
		if err != nil {
			return nil, err
		}

		switch t.Type {
		case TokNow:
			timespec = NewTime(start)

		case TokTomorrow:
			// tomorrow [<time>]
			if timespec == nil {
				timespec = NewTime(now)
			}

			if _, err := timespec.Parse(tokens, TimeAndNumber); err != nil {
				if perr, ok := err.(*ParseError); ok && !perr.EndOfInput() {
					return nil, err
				}
			}

			timespec.Tomorrow()

			break loop

		case TokDay:
			// <day> [<time>]
			day := Days[t.Val]
			today := int(start.Weekday())

			if day < today {
				day += 7
			}

			dist := day - today

			timespec = NewTime(start).AddDays(dist)

			if _, err := timespec.Parse(tokens, TimeAndNumber); err != nil {
				if perr, ok := err.(*ParseError); ok && !perr.EndOfInput() {
					return nil, err
				}
			}

			break loop

		case TokMonth:
			// <month> <day>[<ordinal>] <time> [<year>]
			month := Months[t.Val]
			today := int(start.Month())
			year := int(start.Year())

			if month < today {
				year += 1
			}

			if tn, err := tokens.GetToken(TokNumber); err == nil {
				t, err := time.Parse("2006-01-02 15:04:05.000000000", fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d.%09d", year, month, tn.Num, start.Hour(), start.Minute(), start.Second(), start.Nanosecond()))
				if err != nil {
					return nil, err
				}
				timespec = NewTime(t)
			} else {
				return nil, err
			}

			// consume and discard any ordinal
			tokens.GetToken(TokOrdinal)

			if _, err := timespec.Parse(tokens, TimeOnly); err != nil {
				return nil, err
			}

			if tn, err := tokens.GetToken(TokNumber); err == nil {
				if len(tn.Val) < 4 {
					return nil, &ParseError{
						msg:     "year needs to be four digits",
						invalid: true,
						token:   tn,
					}
				}
				timespec.Year(tn.Num)
			}

			break loop

		case TokDate:
			// <date> [<time>]
			t, err := time.Parse("2006-01-02 15:04:05.000000000", fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d.%09d", t.Year, t.Month, t.Day, start.Hour(), start.Minute(), start.Second(), start.Nanosecond()))
			if err != nil {
				return nil, err
			}

			timespec = NewTime(t)

			if _, err := timespec.Parse(tokens, TimeAndNumber); err != nil {
				if perr, ok := err.(*ParseError); ok && perr.EndOfInput() {
					break loop
				}
				if strings.HasPrefix(err.Error(), "expected time") {
					break loop
				}
				return nil, err
			}

			break loop

		case TokTime:
			// <time> [<tomorrow>]
			timespec = NewTime(start).Hour(t.Hour).Minute(t.Minute)

			if _, err := timespec.ParsePM(tokens); err != nil {
				return nil, err
			}

			if _, err := tokens.GetToken(TokTomorrow); err == nil {
				timespec.Year(now.Year())
				timespec.Month(int(now.Month()))
				timespec.Day(now.Day())
				timespec.Tomorrow()
			}

			break loop

		case TokNumber:
			// <number> [<am|pm>]
			timespec = NewTime(start)

			// handle original timespec - numeric count in hours
			if _, err := tokens.Peek(); err != nil {
				if perr, ok := err.(*ParseError); ok && perr.EndOfInput() {
					timespec.AddHours(t.Num)
					break loop
				}
			}

			timespec.Hour(t.Num).Minute(0)

			_, err := timespec.ParsePM(tokens)
			if err != nil {
				return nil, err
			}

			if _, err := tokens.GetToken(TokTomorrow); err == nil {
				timespec.Year(now.Year())
				timespec.Month(int(now.Month()))
				timespec.Day(now.Day())
				timespec.Tomorrow()
			}

			break loop

		case TokFor:
			fallthrough
		case TokPlus:
			// <plus> <duration> <relspec>
			if timespec == nil {
				timespec = NewTime(start)
			}

			d, err := parseRelativeDuration(tokens)
			if err != nil {
				return nil, err
			}

			timespec.AddMinutes(d)

			break loop

		default:
			return nil, &ParseError{
				msg:     fmt.Sprintf("unknown date/time value: \"%s\" (%s)", t.Val, tokTypes[t.Type]),
				invalid: true,
				token:   t,
			}
		}
	}

	// fmt.Println(now)
	// fmt.Println(start)
	// fmt.Println(timespec)

	return timespec, nil
}

func ParseRange(now time.Time, args []string) (time.Time, time.Time, error) {
	var (
		start    time.Time
		end      time.Time
		timespec time.Time
	)

	tokens, err := tokenize(args)
	if err != nil {
		return start, end, fmt.Errorf("%v", err)
	}

again:
	t, err := tokens.Peek()
	if err != nil {
		return start, end, fmt.Errorf("insufficient timespec in arguments")
	}

	switch t.Type {
	case TokFrom:
		tokens.Pop()
		goto again
	case TokUntil:
		fallthrough
	case TokTo:
		tokens.Pop()
	}

	tval, err := parseTimeSpec(now, now, tokens)
	if err != nil {
		return timespec, end, err
	}

	timespec = tval.Time()

	if timespec.Before(now) {
		return timespec, end, fmt.Errorf("start is in the past")
	}

	if t.Type == TokPlus || t.Type == TokUntil || t.Type == TokTo {
		return now, timespec, nil
	}

	t, err = tokens.Peek()
	if err != nil {
		return now, timespec, nil
	}

	start = timespec

	var separators = map[int]bool{
		TokUntil: true,
		TokTo:    true,
		TokPlus:  true,
		TokFor:   true,
	}

	if _, ok := separators[t.Type]; !ok {
		return start, end, fmt.Errorf("missing separator between start and end")
	}

	if !(t.Type == TokPlus || t.Type == TokFor) {
		tokens.Pop()
	}

	tval, err = parseTimeSpec(now, start, tokens)
	if err != nil {
		return start, end, err
	}

	if t, err := tokens.Peek(); err == nil {
		return start, end, &ParseError{
			msg:     "extra arguments beyond timespec",
			invalid: true,
			token:   t,
		}
	}

	end = tval.Time()
	end = end.Round(time.Minute)

	// fmt.Println(now)
	// fmt.Println(start)
	// fmt.Println(end)

	if end.Before(start) {
		return start, end, fmt.Errorf("end before start")
	}

	return start, end, nil
}

func ParseDuration(now time.Time, args []string) (time.Time, error) {
	var end time.Time

	tokens, err := tokenize(args)
	if err != nil {
		return end, fmt.Errorf("%v", err)
	}

	t, err := tokens.Peek()
	if err != nil {
		return end, fmt.Errorf("insufficient timespec in arguments")
	}

	switch t.Type {
	case TokUntil:
		fallthrough
	case TokTo:
		tokens.Pop()
	}

	tval, err := parseTimeSpec(now, now, tokens)
	if err != nil {
		return end, err
	}

	end = tval.Time()

	if end.Before(now) {
		return end, fmt.Errorf("start is in the past")
	}

	return end, nil
}
