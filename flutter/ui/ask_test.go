package ui_test

import (
	"runtime"
	"testing"

	"gart.dev/ui"
)

// takeCalls reads the queue of the host. The goroutine of the test needs a
// turn of the scheduler before it parks, thus the read runs again.
func takeCalls(t *testing.T) string {
	t.Helper()
	for i := 0; i < 10000; i++ {
		if text := ui.TakeCalls(); text != "" {
			return text
		}
		runtime.Gosched()
	}
	t.Fatal("no request reached the queue of the host")
	return ""
}

func TestCallGivesTheAnswerOfTheHost(t *testing.T) {
	ui.Reset()
	done := make(chan string, 1)
	go func() {
		value, err := ui.Call("share", "a line\nand another\t\"quoted\"")
		if err != nil {
			done <- "error: " + err.Error()
			return
		}
		done <- value
	}()

	text := takeCalls(t)
	// The writer escapes the newline, the tab, and the quotation mark, thus
	// the host reads 1 JSON array with no repair.
	want := `[{"id":1,"name":"share","arg":"a line\nand another\t\"quoted\""}]`
	if text != want {
		t.Fatalf("the queue gave\n%s\nwant\n%s", text, want)
	}
	if again := ui.TakeCalls(); again != "" {
		t.Fatalf("the second read gave %q, want an empty text", again)
	}

	ui.Answer(1, "sent", "")
	if got := <-done; got != "sent" {
		t.Fatalf("the call gave %q, want sent", got)
	}
}

func TestCallGivesTheErrorOfTheHost(t *testing.T) {
	ui.Reset()
	done := make(chan string, 1)
	go func() {
		_, err := ui.Call("datePicker", "")
		if err == nil {
			done <- "no error"
			return
		}
		done <- err.Error()
	}()

	takeCalls(t)
	ui.Answer(1, "", "the user closed the dialog")
	if got := <-done; got != "the user closed the dialog" {
		t.Fatalf("the call gave %q, want the error of the host", got)
	}
}

func TestAnswerOfAnUnknownRequestDoesNothing(t *testing.T) {
	ui.Reset()
	ui.Answer(99, "x", "")
}

func TestTakeCallsGivesAnEmptyTextWithNoRequest(t *testing.T) {
	ui.Reset()
	if text := ui.TakeCalls(); text != "" {
		t.Fatalf("the queue gave %q, want an empty text", text)
	}
}
