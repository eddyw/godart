// Package app holds the whole application. The form validates in Go, and it
// sends the fields to an HTTP API. It reads no Dart name, and a `go test` runs
// each part of it with no Flutter.
package app

import (
	"strings"
	"unicode/utf8"
)

// Fields holds the text of each field of the form.
type Fields struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

// Errors holds the report of each field. An empty string means no error.
type Errors struct {
	Name    string
	Email   string
	Message string
}

// Ok reports whether each field of the form is valid.
func (e Errors) Ok() bool {
	return e.Name == "" && e.Email == "" && e.Message == ""
}

// The limits of the form. The server holds the same 3.
const (
	minName    = 2
	maxName    = 40
	maxMessage = 500
)

// Validate gives the report of each field. It runs on each keystroke, thus it
// reads no network.
func Validate(f Fields) Errors {
	var e Errors
	name := strings.TrimSpace(f.Name)
	switch {
	case name == "":
		e.Name = "the name is empty"
	case utf8.RuneCountInString(name) < minName:
		e.Name = "the name needs 2 characters or more"
	case utf8.RuneCountInString(name) > maxName:
		e.Name = "the name holds more than 40 characters"
	}

	email := strings.TrimSpace(f.Email)
	switch {
	case email == "":
		e.Email = "the address is empty"
	case !validAddress(email):
		e.Email = "the address has no valid form"
	}

	message := strings.TrimSpace(f.Message)
	switch {
	case message == "":
		e.Message = "the message is empty"
	case utf8.RuneCountInString(message) > maxMessage:
		e.Message = "the message holds more than 500 characters"
	}
	return e
}

// validAddress reports whether the text holds 1 local part, 1 at sign, and a
// domain of 2 labels or more. The package net/mail stands outside the subset,
// and a form needs no full grammar of RFC 5322.
func validAddress(s string) bool {
	local, domain, found := strings.Cut(s, "@")
	if !found || local == "" || strings.Contains(domain, "@") {
		return false
	}
	if strings.ContainsAny(s, " \t") {
		return false
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if l == "" {
			return false
		}
	}
	return true
}
