package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/johnstarich/go-wasm/log"
	"mvdan.cc/sh/v3/syntax"
)

type Completion struct {
	Completion string
	Start, End int
}

func getCompletions(line string, cursor int) []Completion {
	completions, err := getCompletionsErr(line, cursor)
	if err != nil {
		log.Error("Failed completions: ", err)
		return nil
	}
	return completions
}

func getCompletionsErr(line string, cursor int) ([]Completion, error) {
	parser := syntax.NewParser()
	var stmts []*syntax.Stmt
	err := parser.Stmts(strings.NewReader(line), func(stmt *syntax.Stmt) bool {
		if int(stmt.Pos().Offset()) <= cursor && int(stmt.End().Offset()) >= cursor {
			stmts = append(stmts, stmt)
		}
		return true
	})
	if err != nil || len(stmts) == 0 {
		return nil, err
	}
	cursorStmt := stmts[0]
	cursorStmtStr := formatStmt(line, cursorStmt)
	cursorStmtOffset := int(cursorStmt.Pos().Offset())
	cursor -= cursorStmtOffset

	var commandWord, cursorWord *syntax.Word
	err = parser.Words(strings.NewReader(cursorStmtStr), func(word *syntax.Word) bool {
		if commandWord == nil {
			commandWord = word
		}
		if int(word.Pos().Offset()) <= cursor && int(word.End().Offset()) >= cursor {
			cursorWord = word
		}
		return true
	})
	if err != nil || cursorWord == nil {
		return nil, err
	}

	commandWordStr, err := evalWord(commandWord.Parts)
	if err != nil {
		return nil, err
	}
	cursorWordStr, err := evalWord(cursorWord.Parts)
	if err != nil {
		return nil, err
	}

	return getStatementCompletions(
		commandWordStr,
		cursorWordStr,
		cursorStmtOffset+int(cursorWord.Pos().Offset()),
		cursorStmtOffset+int(cursorWord.End().Offset()))
}

func getStatementCompletions(commandName string, word string, start, end int) ([]Completion, error) {
	switch {
	case strings.Contains(word, "/"):
		dir := word
		filter := false
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			dir = filepath.Dir(dir)
			filter = true
		}
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			return nil, nil
		}
		var completions []Completion
		for _, d := range dirEntries {
			base := filepath.Base(word)
			name := d.Name()
			if !filter || strings.HasPrefix(name, base) {
				file := fileJoin(dir, name)
				if d.IsDir() {
					file += string(filepath.Separator)
				}
				completions = append(completions, Completion{
					Completion: file,
					Start:      start,
					End:        end,
				})
			}
		}
		return completions, nil
	default:
		return nil, nil
	}
}

func fileJoin(a, b string) string {
	if a == "." {
		return "." + string(filepath.Separator) + b
	}
	return filepath.Join(a, b)
}
