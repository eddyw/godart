package ui

import (
	"errors"
	"strconv"
	"strings"
)

// This file holds the calls that a Go program makes into the host of Flutter.
// Each 1 stands on Call of ask.go, thus each 1 must run in a goroutine.
//
// The host of goflutter answers each name below with no code of the
// application. A handler that goSetCall installs can answer a name of the
// application first, and it then gives each other name to goHostCall.

// DateRange asks the host for a range of 2 days.
//
// Each argument and each result holds the milliseconds of the Unix epoch. A
// first of 0 names the day of today, and a last of 0 names the day 365 days
// after first. A start and an end of 0 open the dialog with no selection.
//
// The 2 results are 0 when the user closed the dialog.
//
// The call needs a goroutine and an Anchor node in the tree.
func DateRange(first, last, start, end int) (int, int, error) {
	arg := strconv.Itoa(first) + "," + strconv.Itoa(last) + "," +
		strconv.Itoa(start) + "," + strconv.Itoa(end)
	out, err := Call("dateRange", arg)
	if err != nil {
		return 0, 0, err
	}
	a, b, ok := strings.Cut(out, ",")
	if !ok {
		return 0, 0, nil
	}
	from, err := strconv.Atoi(a)
	if err != nil {
		return 0, 0, errors.New("the host gave the date range " + out)
	}
	to, err := strconv.Atoi(b)
	if err != nil {
		return 0, 0, errors.New("the host gave the date range " + out)
	}
	return from, to, nil
}

// Share gives a text to the share sheet of the platform.
//
// The host of goflutter holds no such sheet, because a share needs a plugin.
// An application that names share_plus answers this call in its own handler.
//
// The call needs a goroutine.
func Share(text string) error {
	_, err := Call("share", text)
	return err
}

// SnackBar shows 1 message over the application.
//
// An empty action gives a bar with no button. The answer is "action" when the
// user pressed the button, and an empty text in each other case.
//
// The call needs a goroutine and an Anchor node in the tree.
func SnackBar(message, action string) (string, error) {
	return Call("snackBar", pair(message, action))
}

// SaveValue writes 1 value that survives a restart of the application.
//
// The host of goflutter holds the value in memory. An application that names
// shared_preferences answers this call in its own handler.
//
// The call needs a goroutine.
func SaveValue(key, value string) error {
	_, err := Call("store.set", pair(key, value))
	return err
}

// ReadValue reads 1 value that SaveValue wrote. An absent key gives an empty
// text.
//
// The call needs a goroutine.
func ReadValue(key string) (string, error) { return Call("store.get", key) }

// pair writes 2 texts as a JSON array. The host reads that array.
func pair(a, b string) string {
	var out strings.Builder
	out.WriteByte('[')
	writeJSONText(&out, a)
	out.WriteByte(',')
	writeJSONText(&out, b)
	out.WriteByte(']')
	return out.String()
}
