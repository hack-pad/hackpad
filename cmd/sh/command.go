package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/syntax"
)

func runLine(term console.Console, line string) error {
	parser := syntax.NewParser()
	var cmdErr error
	err := parser.Stmts(strings.NewReader(line), func(stmt *syntax.Stmt) bool {
		cmdErr = runCommand(term, line, stmt)
		return cmdErr == nil
	})
	if err != nil {
		return err
	}
	if cmdErr != nil {
		return cmdErr
	}
	if parser.Incomplete() {
		return errors.New("Incomplete command. Multi-line commands not supported.")
	}
	return nil
}

func tokenize(line string) []string {
	var tokens []string
	var buf []rune
	for _, c := range line {
		if unicode.IsSpace(c) && len(buf) != 0 {
			tokens = append(tokens, string(buf))
			buf = nil
		} else {
			buf = append(buf, c)
		}
	}
	if len(buf) != 0 {
		tokens = append(tokens, string(buf))
	}
	return tokens
}

func evalWord(parts []syntax.WordPart) (string, error) {
	s := ""
	for _, part := range parts {
		switch part := part.(type) {
		case *syntax.Lit:
			s += part.Value
		case *syntax.SglQuoted:
			if part.Dollar {
				return "", errors.Errorf("Dollar single-quotes not supported: %v", part)
			}
			s += part.Value
		case *syntax.DblQuoted:
			if part.Dollar {
				return "", errors.Errorf("Dollar single-quotes not supported: %v", part)
			}
			dblQuoted, err := evalWord(part.Parts)
			if err != nil {
				return "", err
			}
			s += dblQuoted
		case *syntax.ParamExp:
			name := part.Param.Value
			if part.Excl || part.Length || part.Width || part.Index != nil || part.Slice != nil || part.Repl != nil || part.Names != 0 || part.Exp != nil {
				return "", errors.Errorf("Variable expansion type not supported: %s %v", name, part)
			}
			s += os.Getenv(name)
		case *syntax.CmdSubst, *syntax.ArithmExp, *syntax.ProcSubst, *syntax.ExtGlob:
			return "", errors.Errorf("Unrecognized word part type: %T %v", part, part)
		default:
			return "", errors.Errorf("Unrecognized word part type: %T %v", part, part)
		}
	}
	return s, nil
}

func runCommand(term console.Console, line string, stmt *syntax.Stmt) error {
	switch node := stmt.Cmd.(type) {
	case *syntax.CallExpr:
		var env []string
		for _, assign := range node.Assigns {
			key := assign.Name.Value
			value, err := evalWord(assign.Value.Parts)
			if err != nil {
				return err
			}
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}

		var args []string
		for _, arg := range node.Args {
			argStr, err := evalWord(arg.Parts)
			if err != nil {
				return err
			}
			args = append(args, argStr)
		}
		if len(args) == 0 {
			return errors.New("Setting variables only is not supported")
		}

		if len(stmt.Redirs) > 0 {
			return errors.New("File redirects are not supported")
		}

		commandName, args := args[0], args[1:]
		cmd := exec.Command(commandName, args...)
		cmd.Env = append(os.Environ(), env...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = term.Stdout()
		cmd.Stderr = term.Stderr()
		return runCmd(cmd, cmdOptions{
			Background: stmt.Background,
			Pipe:       false,
		})

	case *syntax.IfClause, *syntax.WhileClause, *syntax.ForClause, *syntax.CaseClause, *syntax.Block, *syntax.Subshell, *syntax.FuncDecl, *syntax.ArithmCmd, *syntax.TestClause, *syntax.DeclClause, *syntax.LetClause, *syntax.TimeClause, *syntax.CoprocClause, *syntax.BinaryCmd:
		return errors.Errorf("Unknown statement type: %T %v", stmt.Cmd, stmt.Cmd)
	default:
		return errors.Errorf("Unknown statement type: %T %v", stmt.Cmd, stmt.Cmd)
	}
}

type cmdOptions struct {
	Background bool
	Pipe       bool
}

func runCmd(cmd *exec.Cmd, options cmdOptions) error {
	args := []string{cmd.Path}
	if len(cmd.Args) > 0 {
		args = cmd.Args
	}
	commandName, args := args[0], args[1:]

	builtin, isBuiltin := builtins[commandName]
	if options.Pipe || !isBuiltin {
		if options.Background {
			return cmd.Start()
		} else {
			return cmd.Run()
		}
	}

	var oldKV, unsetKV []string
	// override env for builtin
	for _, pair := range cmd.Env {
		key, value := splitKeyValue(pair)
		if oldValue, isSet := os.LookupEnv(key); isSet {
			oldKV = append(oldKV, key+"="+oldValue)
		} else {
			unsetKV = append(unsetKV, key)
		}
		os.Setenv(key, value)
	}
	err := builtin(&writerConsole{stdout: cmd.Stdout, stderr: cmd.Stderr}, args...)
	// restore env
	for _, pair := range oldKV {
		key, value := splitKeyValue(pair)
		os.Setenv(key, value)
	}
	for _, key := range unsetKV {
		os.Unsetenv(key)
	}
	return errors.Wrap(err, commandName)
}

type writerConsole struct {
	stdout, stderr io.Writer
}

func (c *writerConsole) Stdout() io.Writer {
	return &carriageReturnWriter{c.stdout}
}

func (c *writerConsole) Stderr() io.Writer {
	return &carriageReturnWriter{c.stderr}
}

func (c *writerConsole) Note() io.Writer {
	return ioutil.Discard
}
