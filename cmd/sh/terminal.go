package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime/debug"
	"unicode"

	"github.com/fatih/color"
	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
)

const (
	controlBackspace  = '\x7F'
	controlDeleteWord = '\x17'
	controlEnter      = '\r'
	escapeCSI         = '\x1B'
	escapeLBracket    = '['
)

const (
	prompt = "$ "
)

type terminal struct {
	line   []rune
	cursor int
}

func newTerminal() *terminal {
	return &terminal{}
}

func (t *terminal) Stdout() io.Writer {
	return &carriageReturnWriter{os.Stdout}
}

func (t *terminal) Stderr() io.Writer {
	return &carriageReturnWriter{os.Stderr}
}

func (t *terminal) Note() io.Writer {
	return ioutil.Discard
}

func (t *terminal) Print(args ...interface{}) {
	fmt.Fprint(t.Stdout(), args...)
}

func (t *terminal) Printf(format string, args ...interface{}) {
	fmt.Fprintf(t.Stdout(), format, args...)
}

func (t *terminal) ErrPrint(args ...interface{}) {
	fmt.Fprint(t.Stderr(), args...)
}

func (t *terminal) ReadEvalPrintLoop(reader io.RuneReader) {
	fmt.Fprint(t.Stdout(), prompt)
	for {
		t.ReadEvalPrint(reader)
	}
}

func (t *terminal) ReadEvalPrint(reader io.RuneReader) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("\n\npanic: %s\n%s\n", r, string(debug.Stack()))
			t.ErrPrint(color.RedString(msg))

			// attempt to return to a recovered state
			t.line = nil
			t.cursor = 0
			t.Print(prompt)
		}
	}()

	r, _, err := reader.ReadRune()
	if err != nil {
		log.Error("Error reading from stdin:", err)
	}

	switch r {
	case escapeCSI:
		err := t.ReadEvalEscape(r, reader)
		if err != nil {
			log.Error("Error reading from stdin:", err)
		}
	case controlBackspace:
		if t.cursor > 0 {
			t.cursor--
			prefix, suffix := splitRunes(t.line, t.cursor)
			suffix = suffix[1:] // trim off char after decremented cursor
			t.line = append(prefix, suffix...)
			t.CursorLeftN(1)
			t.ClearRightN(len(t.line) - t.cursor + 1)
			t.Print(string(t.line[t.cursor:]))
			t.CursorLeftN(len(t.line) - t.cursor)
		}
	case controlEnter:
		t.Print("\r\n")
		command := string(t.line)
		t.line = nil
		t.cursor = 0
		err := runCommand(t, command)
		if err != nil {
			t.ErrPrint(color.RedString(err.Error()) + "\n")
		}
		t.Print(prompt)
	case controlDeleteWord,
		'\t': // ignore for now
	default:
		prefix, suffix := splitRunes(t.line, t.cursor)
		t.cursor++
		t.line = append(append(prefix, r), suffix...)
		t.Print(string(t.line[t.cursor-1:]))
		t.CursorLeftN(len(t.line) - t.cursor)
	}
	log.Debugf("Term = %q %d; Cursor = %q %d", string(t.line), len(t.line), string(t.line[t.cursor:]), t.cursor)
}

func splitRunes(runes []rune, i int) (a, b []rune) {
	a = append([]rune{}, runes[:i]...)
	b = append([]rune{}, runes[i:]...)
	return
}

func (t *terminal) ReadEvalEscape(firstRune rune, r io.RuneReader) error {
	controlRune, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	switch controlRune {
	case controlBackspace: // ignore for now
		return nil
	case escapeLBracket:
	default:
		t.Print(string(controlRune))
		return errors.Errorf(`Invalid escape sequence: \x%x \x%x`, escapeCSI, controlRune)
	}

	var controlParams []rune
	for {
		controlRune, _, err = r.ReadRune()
		if err != nil {
			return err
		}
		if !unicode.IsDigit(controlRune) && controlRune != ';' {
			break
		}
		controlParams = append(controlParams, controlRune)
	}

	escape := append(append([]rune{escapeCSI, escapeLBracket}, controlParams...), controlRune)
	log.Printf("Got escape sequence: %q", escape)
	switch controlRune {
	case 'A': // cursor up
		return nil
	case 'B': // cursor down
		return nil
	case 'C': // cursor forward
		if t.cursor >= len(t.line) {
			return nil
		}
		t.cursor++
	case 'D': // cursor backward
		if t.cursor <= 0 {
			return nil
		}
		t.cursor--
	case 'E': // cursor next line
		return nil
	case 'F': // cursor backward
		return nil
	case '~': // forward delete
		if t.cursor != len(t.line) {
			prefix, suffix := splitRunes(t.line, t.cursor)
			suffix = suffix[1:]
			t.line = append(prefix, suffix...)
			t.ClearRightN(len(t.line) - t.cursor + 1)
			t.Print(string(t.line[t.cursor:]))
			t.CursorLeftN(len(t.line) - t.cursor)
		}
		return nil
	default:
		// ignore by default
		return nil
	}
	t.Print(string(escape))
	return nil
}

func (t *terminal) ClearRightN(n int) {
	if n <= 0 {
		return
	}
	t.Printf("%c%c%dX", escapeCSI, escapeLBracket, n)
}

func (t *terminal) CursorLeftN(n int) {
	if n <= 0 {
		return
	}
	t.Printf("%c%c%dD", escapeCSI, escapeLBracket, n)
}
