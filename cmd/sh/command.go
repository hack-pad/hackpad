package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/syntax"
)

func runLine(term console.Console, line string) error {
	parser := syntax.NewParser()
	var cmdErr error
	err := parser.Stmts(strings.NewReader(line), func(stmt *syntax.Stmt) bool {
		cmdErr = runCommand(term, line, stmt, false)
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

func runCommand(term console.Console, line string, stmt *syntax.Stmt, isPipe bool) error {
	switch node := stmt.Cmd.(type) {
	case *syntax.CallExpr:
		return runCallExpr(term, stmt, node, isPipe)
	case *syntax.BinaryCmd:
		switch node.Op {
		case syntax.AndStmt: // &&
			err := runCommand(term, line, node.X, false)
			if err != nil {
				return err
			}
			return runCommand(term, line, node.Y, false)
		case syntax.OrStmt: // ||
			err := runCommand(term, line, node.X, false)
			if err == nil {
				return nil
			}
			return runCommand(term, line, node.Y, false)
		case syntax.Pipe: // |
			r, w, err := os.Pipe()
			if err != nil {
				return err
			}
			leftTerm := &redirectConsole{
				stdin:  getConsoleStdin(term),
				stdout: w,
				stderr: term.Stderr(),
			}
			rightTerm := &redirectConsole{
				stdin:  r,
				stdout: term.Stdout(),
				stderr: term.Stderr(),
			}
			errChan := make(chan error, 1)
			go func() {
				errChan <- runCommand(rightTerm, line, node.Y, true)
			}()
			err = runCommand(leftTerm, line, node.X, false)
			if err != nil {
				return err
			}
			w.Close()
			return <-errChan
		case syntax.PipeAll: // |&
			r, w, err := os.Pipe()
			if err != nil {
				return err
			}
			leftTerm := &redirectConsole{
				stdin:  getConsoleStdin(term),
				stdout: w,
				stderr: w,
			}
			rightTerm := &redirectConsole{
				stdin:  r,
				stdout: term.Stdout(),
				stderr: term.Stderr(),
			}
			errChan := make(chan error, 1)
			go func() {
				errChan <- runCommand(rightTerm, line, node.Y, true)
			}()
			err = runCommand(leftTerm, line, node.X, false)
			if err != nil {
				return err
			}
			return <-errChan
		default:
			return errors.Errorf("Unknown binary operator: %v", node.Op)
		}

	case *syntax.TimeClause:
		start := time.Now()
		err := runCommand(term, line, node.Stmt, false)
		duration := time.Since(start)
		fmt.Fprintf(term.Stdout(), "\n%s\t %v total\n", formatStmt(line, node.Stmt), duration)
		return err

	case *syntax.IfClause, *syntax.WhileClause, *syntax.ForClause, *syntax.CaseClause, *syntax.Block, *syntax.Subshell, *syntax.FuncDecl, *syntax.ArithmCmd, *syntax.TestClause, *syntax.DeclClause, *syntax.LetClause, *syntax.CoprocClause:
		return errors.Errorf("Unimplemented statement type: %T %v", stmt.Cmd, stmt.Cmd)
	default:
		return errors.Errorf("Unknown statement type: %T %v", stmt.Cmd, stmt.Cmd)
	}
}

func formatStmt(source string, s *syntax.Stmt) string {
	return source[s.Pos().Offset():s.End().Offset()]
}

type cmdOptions struct {
	Background bool
	Pipe       bool
}

func runCmd(cmd *exec.Cmd, options cmdOptions) error {
	// ensure files are all attached by default. these are assumed to be set up already
	if cmd.Stdin == nil || cmd.Stdout == nil || cmd.Stderr == nil {
		panic("Standard files not set up")
	}

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
	err := builtin(&redirectConsole{
		stdin:  cmd.Stdin,
		stdout: cmd.Stdout,
		stderr: cmd.Stderr,
	}, args...)
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

type redirectConsole struct {
	stdin          io.Reader
	stdout, stderr io.Writer
}

func (c *redirectConsole) Stdin() io.Reader {
	return c.stdin
}

func (c *redirectConsole) Stdout() io.Writer {
	return &carriageReturnWriter{c.stdout}
}

func (c *redirectConsole) Stderr() io.Writer {
	return &carriageReturnWriter{c.stderr}
}

func (c *redirectConsole) Note() io.Writer {
	return ioutil.Discard
}

func getConsoleStdin(term console.Console) io.Reader {
	if stdiner, ok := term.(interface{ Stdin() io.Reader }); ok {
		return stdiner.Stdin()
	}
	return os.Stdin
}
