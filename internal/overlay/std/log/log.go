//go:build dart

// Package log writes a line to a writer, with a prefix and a stamp of the
// clock.
//
// The flags Llongfile and Lshortfile stay outside the subset. Each one names
// the source position of the caller, and that position needs runtime.Caller.
// A call that sets one of the 2 stops the program.
package log

import (
	"fmt"
	"io"
	"os"
	"time"
)

// The flags name each part that the header of a line holds.
const (
	Ldate         = 1 << iota // the date of the local zone: 2009/01/23
	Ltime                     // the clock of the local zone: 01:23:23
	Lmicroseconds             // the microsecond of the clock: 01:23:23.123123
	Llongfile                 // the full path and the line: /a/b/c/d.go:23
	Lshortfile                // the base name and the line: d.go:23
	LUTC                      // the date and the clock of UTC
	Lmsgprefix                // the prefix stands in front of the message
	LstdFlags     = Ldate | Ltime
)

// A Logger writes a line to a writer. Each write holds 1 line.
type Logger struct {
	out       io.Writer
	prefix    string
	flag      int
	isDiscard bool
}

// New builds a Logger that writes to out.
func New(out io.Writer, prefix string, flag int) *Logger {
	rejectFileFlag(flag)
	return &Logger{out: out, prefix: prefix, flag: flag}
}

// Default gives the logger of the top level functions of the package.
func Default() *Logger { return std }

var std = New(os.Stderr, "", LstdFlags)

// rejectFileFlag stops the program for a flag that names a source position.
func rejectFileFlag(flag int) {
	if flag&(Llongfile|Lshortfile) != 0 {
		panic("log: the flags Llongfile and Lshortfile need runtime.Caller, " +
			"and that function stays outside the subset")
	}
}

// header writes the prefix and the stamp of the clock.
func (l *Logger) header(t time.Time) string {
	out := ""
	if l.flag&Lmsgprefix == 0 {
		out = l.prefix
	}
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if l.flag&LUTC != 0 {
			t = t.UTC()
		}
		if l.flag&Ldate != 0 {
			out += t.Format("2006/01/02") + " "
		}
		if l.flag&(Ltime|Lmicroseconds) != 0 {
			if l.flag&Lmicroseconds != 0 {
				out += t.Format("15:04:05.000000") + " "
			} else {
				out += t.Format("15:04:05") + " "
			}
		}
	}
	if l.flag&Lmsgprefix != 0 {
		out += l.prefix
	}
	return out
}

// Output writes 1 line. The depth of the call reaches no operation, because
// the subset holds no source position of a caller.
func (l *Logger) Output(calldepth int, s string) error {
	if l.out == nil {
		return nil
	}
	line := l.header(time.Now()) + s
	if len(line) == 0 || line[len(line)-1] != '\n' {
		line += "\n"
	}
	_, err := l.out.Write([]byte(line))
	return err
}

func (l *Logger) Print(v ...any)                 { l.Output(2, fmt.Sprint(v...)) }
func (l *Logger) Printf(format string, v ...any) { l.Output(2, fmt.Sprintf(format, v...)) }
func (l *Logger) Println(v ...any)               { l.Output(2, fmt.Sprintln(v...)) }

func (l *Logger) Fatal(v ...any) {
	l.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...any) {
	l.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *Logger) Fatalln(v ...any) {
	l.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *Logger) Panic(v ...any) {
	s := fmt.Sprint(v...)
	l.Output(2, s)
	panic(s)
}

func (l *Logger) Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	l.Output(2, s)
	panic(s)
}

func (l *Logger) Panicln(v ...any) {
	s := fmt.Sprintln(v...)
	l.Output(2, s)
	panic(s)
}

func (l *Logger) SetOutput(w io.Writer)   { l.out = w }
func (l *Logger) Writer() io.Writer       { return l.out }
func (l *Logger) Prefix() string          { return l.prefix }
func (l *Logger) SetPrefix(prefix string) { l.prefix = prefix }
func (l *Logger) Flags() int              { return l.flag }

func (l *Logger) SetFlags(flag int) {
	rejectFileFlag(flag)
	l.flag = flag
}

// The functions below write through the logger of the package.

func Print(v ...any)                 { std.Output(2, fmt.Sprint(v...)) }
func Printf(format string, v ...any) { std.Output(2, fmt.Sprintf(format, v...)) }
func Println(v ...any)               { std.Output(2, fmt.Sprintln(v...)) }

func Fatal(v ...any) {
	std.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func Fatalf(format string, v ...any) {
	std.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func Fatalln(v ...any) {
	std.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}

func Panic(v ...any) {
	s := fmt.Sprint(v...)
	std.Output(2, s)
	panic(s)
}

func Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	std.Output(2, s)
	panic(s)
}

func Panicln(v ...any) {
	s := fmt.Sprintln(v...)
	std.Output(2, s)
	panic(s)
}

func Output(calldepth int, s string) error { return std.Output(calldepth+1, s) }
func SetOutput(w io.Writer)                { std.SetOutput(w) }
func Writer() io.Writer                    { return std.Writer() }
func Flags() int                           { return std.Flags() }
func SetFlags(flag int)                    { std.SetFlags(flag) }
func Prefix() string                       { return std.Prefix() }
func SetPrefix(prefix string)              { std.SetPrefix(prefix) }
