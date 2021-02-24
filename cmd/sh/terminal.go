package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
)

const (
	controlBackspace      = '\x7F'
	controlClear          = '\f'
	controlCloseStdin     = '\x04'
	controlSigTStop       = '\x03'
	controlCursorBackward = 'D'
	controlCursorDown     = 'B'
	controlCursorForward  = 'C'
	controlCursorUp       = 'A'
	controlDeleteWord     = '\x17'
	controlEnd            = '\x05'
	controlEnter          = '\r'
	controlHome           = '\x01'
	controlNextWord       = '\x66'
	controlPreviousWord   = '\x62'
	controlScroll         = '\x4f'
	escapeCSI             = '\x1B'
	escapeLBracket        = '['
)

type terminal struct {
	// reader state
	line   []rune
	cursor int
	// command state
	lastExitCode int
	history      *history
}

func newTerminal() *terminal {
	term := &terminal{}
	history, err := newHistory()
	if err != nil {
		term.ErrPrint(color.RedString(err.Error()) + "\n")
	}
	term.history = history
	return term
}

func (t *terminal) Stdout() io.Writer {
	return os.Stdout
}

func (t *terminal) Stderr() io.Writer {
	return os.Stderr
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

func (t *terminal) ReadEvalPrintLoop(reader io.RuneReader) int {
	fmt.Fprint(t.Stdout(), prompt(t))
	for {
		err := t.ReadEvalPrint(reader)
		if exitErr, ok := err.(*ExitErr); ok {
			return exitErr.Code
		}
		if err == io.EOF || unwrapErr(err) == os.ErrClosed {
			return 0
		}
		if err != nil {
			log.Error("Critical error during REPL: ", err)
			return 1
		}
	}
}

func unwrapErr(err error) error {
	for {
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return err
		}
		newErr := unwrapper.Unwrap()
		if newErr != nil {
			err = newErr
		}
	}
}

func (t *terminal) ReadEvalPrint(reader io.RuneReader) error {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("\n\npanic: %s\n%s\n", r, string(debug.Stack()))
			t.ErrPrint(color.RedString(msg))

			// attempt to return to a recovered state
			t.line = nil
			t.cursor = 0
			t.lastExitCode = 1
			t.Print(prompt(t))
		}
	}()

	r, _, err := reader.ReadRune()
	if err == io.EOF {
		return io.EOF
	}
	if err != nil {
		return err
	}

	switch r {
	case escapeCSI:
		err := t.ReadEvalEscape(r, reader)
		if err != nil {
			log.Error("Error reading from stdin: ", err)
		}
	case controlBackspace:
		if t.cursor > 0 {
			t.cursor--
			runes, suffix := splitRunes(t.line, t.cursor)
			suffix = suffix[1:] // trim off char after decremented cursor
			runes = append(runes, suffix...)
			t.line = runes
			t.CursorLeftN(1)
			t.ClearRightN(len(t.line) - t.cursor + 1)
			t.Print(string(t.line[t.cursor:]))
			t.CursorLeftN(len(t.line) - t.cursor)
		}
	case controlClear:
		t.Clear()
		t.Print(prompt(t))
	case controlEnter:
		t.Print("\r\n")
		command := string(t.line)
		t.line = nil
		t.cursor = 0
		err = runLine(t, command)
		t.lastExitCode = 0
		if err != nil {
			t.ErrPrint(color.RedString(err.Error()) + "\n")
			t.lastExitCode = 1
			if exitErr, ok := err.(*exec.ExitError); ok {
				t.lastExitCode = exitErr.ExitCode()
			}
		}
		err := t.history.Push(command)
		if err != nil {
			t.ErrPrint(color.RedString(err.Error()) + "\n")
		}
		t.Print(prompt(t))
	case controlDeleteWord:
		t.deleteWord()
	case controlEnd:
		t.moveCursorToEnd()
	case controlHome:
		t.moveCursorToStart()
	case controlCloseStdin:
		return &ExitErr{Code: 0}
	case controlSigTStop:
		t.line = nil
		t.cursor = 0
		err := t.history.Push("") // resets history index, no error should be possible
		if err != nil {
			panic(err)
		}
		t.lastExitCode = 1
		t.Print("^C\n\r")
		t.Print(prompt(t))
	case '\t': // ignore for now
	default:
		prefix, suffix := splitRunes(t.line, t.cursor)
		t.cursor++
		t.line = append(append(prefix, r), suffix...)
		t.Print(string(t.line[t.cursor-1:]))
		t.CursorLeftN(len(t.line) - t.cursor)
	}
	if t.cursor > len(t.line) {
		panic(fmt.Sprint("Cursor too large: cursor =", t.cursor, "length =", len(t.line)))
	}
	log.Debugf("Term = %q %d; Cursor = %q %d", string(t.line), len(t.line), string(t.line[t.cursor:]), t.cursor)
	return nil
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
	case controlBackspace:
		t.deleteWord()
		return nil
	case controlPreviousWord:
		beforeCursor := string(t.line[:t.cursor])
		beforeCursor = strings.TrimRightFunc(beforeCursor, unicode.IsSpace)
		prevWord := strings.LastIndexFunc(beforeCursor, unicode.IsSpace) + 1
		t.CursorLeftN(t.cursor - prevWord)
		t.cursor = prevWord
		return nil
	case controlNextWord:
		afterCursor := string(t.line[t.cursor:])
		afterCursor = strings.TrimLeftFunc(afterCursor, func(r rune) bool {
			return !unicode.IsSpace(r)
		})
		afterCursor = strings.TrimLeftFunc(afterCursor, unicode.IsSpace)
		nextWord := len(t.line) - len(afterCursor)
		t.CursorRightN(nextWord - t.cursor)
		t.cursor = nextWord
		return nil
	case controlScroll: // ignore: this is a pre-cursor to a scroll event, but unsure if it should have a special action
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
	log.Debugf("Got escape sequence: %q", escape)
	switch controlRune {
	case controlCursorUp:
		previousCommand, ok := t.history.Previous()
		if ok {
			t.CursorLeftN(t.cursor)
			t.ClearRightN(len(t.line))
			t.line = []rune(previousCommand)
			t.cursor = len(t.line)
			t.Print(previousCommand)
		}
		return nil
	case controlCursorDown:
		nextCommand, _ := t.history.Next()
		t.CursorLeftN(t.cursor)
		t.ClearRightN(len(t.line))
		t.line = []rune(nextCommand)
		t.cursor = len(t.line)
		t.Print(nextCommand)
		return nil
	case controlCursorForward:
		if t.cursor >= len(t.line) {
			return nil
		}
		t.cursor++
	case controlCursorBackward:
		if t.cursor <= 0 {
			return nil
		}
		t.cursor--
	case 'E': // cursor next line
		return nil
	case 'F': // end key (also cursor backward?)
		t.moveCursorToEnd()
		return nil
	case 'H': // home key
		t.moveCursorToStart()
		return nil
	case '~': // forward delete
		if t.cursor != len(t.line) {
			runes, suffix := splitRunes(t.line, t.cursor)
			suffix = suffix[1:]
			runes = append(runes, suffix...)
			t.line = runes
			t.ClearRightN(len(t.line) - t.cursor + 1)
			t.Print(string(t.line[t.cursor:]))
			t.CursorLeftN(len(t.line) - t.cursor)
		}
		return nil
	default:
		// ignore by default
		return nil
	}
	str := string(escape)
	t.Print(str)
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
	t.Printf("%c%c%d%c", escapeCSI, escapeLBracket, n, controlCursorBackward)
}

func (t *terminal) CursorRightN(n int) {
	if n <= 0 {
		return
	}
	t.Printf("%c%c%d%c", escapeCSI, escapeLBracket, n, controlCursorForward)
}

func (t *terminal) Clear() {
	// TODO this wipes out some scrollback, need to figure out how to preserve it
	t.Print(string(escapeCSI) + "[H") // set cursor to top left
	t.Print(string(escapeCSI) + "[J") // clear viewport
}

func (t *terminal) deleteWord() {
	originalLen := len(t.line)
	var trimmed []rune
	t.line, trimmed = deleteWord(t.line, t.cursor)
	trimmedLen := len(trimmed)
	t.cursor -= trimmedLen
	t.CursorLeftN(trimmedLen)
	t.ClearRightN(originalLen - t.cursor)
	remaining := t.line[t.cursor:]
	t.Print(string(remaining))
	t.CursorLeftN(len(remaining))
}

func deleteWord(s []rune, cursor int) (newLine, trimmed []rune) {
	if cursor == 0 {
		return s, nil
	}

	str := string(s[:cursor])
	str = strings.TrimRightFunc(str, unicode.IsSpace)
	previousWord := strings.LastIndexFunc(str, unicode.IsSpace) + 1
	// not found is: -1 + 1 == 0
	// finding a word is: lastSpaceIndex + 1

	newS := string(s[:previousWord]) + string(s[cursor:])
	return []rune(newS), s[previousWord:cursor]
}

func (t *terminal) moveCursorToStart() {
	t.CursorLeftN(t.cursor)
	t.cursor = 0
}

func (t *terminal) moveCursorToEnd() {
	t.CursorRightN(len(t.line) - t.cursor)
	t.cursor = len(t.line)
}
