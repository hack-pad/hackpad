package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
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

func runCommand(term console.Console, line string, stmt *syntax.Stmt, isPipe bool) error {
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

		commandName, args := args[0], args[1:]
		cmd := exec.Command(commandName, args...)
		cmd.Env = append(os.Environ(), env...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = term.Stdout()
		cmd.Stderr = term.Stderr()

		for _, redir := range stmt.Redirs {
			var redirectPtr string
			var err error
			switch redir.Op {
			case syntax.Hdoc, // <<
				syntax.DashHdoc: // <<-
				if redir.Hdoc == nil {
					var word string
					if redir.Word != nil {
						word, _ = evalWord(redir.Word.Parts)
						word = ": " + word
					}
					return errors.New("Invalid heredoc" + word)
				}
				redirectPtr, err = evalWord(redir.Hdoc.Parts)
			case syntax.WordHdoc: // <<<
				redirectPtr, err = evalWord(redir.Word.Parts)
			default:
				redirectPtr, err = evalWord(redir.Word.Parts)
			}
			if err != nil {
				return err
			}

			var fd int
			switch redir.Op {
			case syntax.RdrOut, // >
				syntax.AppOut, // >>
				syntax.RdrAll, // &>
				syntax.AppAll: // &>>
				fd = 1
			default:
				fd = 0
			}
			if redir.N != nil {
				fdStr := redir.N.Value
				parsedFD, err := strconv.ParseUint(fdStr, 10, 64)
				if err != nil {
					return err
				}
				fd = int(parsedFD)
			}

			switch redir.Op {
			case syntax.RdrOut, // >
				syntax.AppOut, // >>
				syntax.RdrAll, // &>
				syntax.AppAll: // &>>
				if fd == 0 {
					return errors.New("Can't redirect stdin to an output file")
				}

				flag := os.O_WRONLY | os.O_CREATE
				if redir.Op == syntax.AppOut || redir.Op == syntax.AppAll {
					flag |= os.O_APPEND
				} else {
					flag |= os.O_TRUNC
				}
				file, err := os.OpenFile(redirectPtr, flag, 0700)
				if err != nil {
					return err
				}

				switch fd {
				case 1:
					cmd.Stdout = file
				case 2:
					cmd.Stderr = file
				default:
					cmd.ExtraFiles = append(cmd.ExtraFiles, file)
				}
			case syntax.RdrIn: // <
				if fd != 0 {
					return errors.New("Can't redirect non-stdin to an input file")
				}

				file, err := os.OpenFile(redirectPtr, os.O_RDONLY, 0)
				if err != nil {
					return err
				}

				cmd.Stdin = file
			case syntax.Hdoc, // <<
				syntax.DashHdoc, // <<-
				syntax.WordHdoc: // <<<
				file := strings.NewReader(redirectPtr)
				if fd != 0 {
					return errors.New("Can't redirect non-stdin to an input file")
				}
				cmd.Stdin = file
			default:
				return errors.Errorf("File redirect of type %q are not supported", redir.Op.String())
			}
		}

		err := runCmd(cmd, cmdOptions{
			Background: stmt.Background,
			Pipe:       isPipe,
		})
		err = exitErrFromCmd(err, stmt.Negated)
		return err

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
				stdin:  os.Stdin,
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
				stdin:  os.Stdin,
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

	case *syntax.IfClause, *syntax.WhileClause, *syntax.ForClause, *syntax.CaseClause, *syntax.Block, *syntax.Subshell, *syntax.FuncDecl, *syntax.ArithmCmd, *syntax.TestClause, *syntax.DeclClause, *syntax.LetClause, *syntax.TimeClause, *syntax.CoprocClause:
		return errors.Errorf("Unknown statement type: %T %v", stmt.Cmd, stmt.Cmd)
	default:
		return errors.Errorf("Unknown statement type: %T %v", stmt.Cmd, stmt.Cmd)
	}
}

func exitErrFromCmd(err error, negated bool) error {
	code := exitCodeFromCmd(err, negated)
	if code == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("Negated return code")
}

// exitCodeFromCmd tries to produce an exit code for the given error.
// 0 for success, non-0 for failure.
// If negated is true, the success result is flipped.
func exitCodeFromCmd(err error, negated bool) int {
	return negateExitCode(exitCodeFromErr(err), negated)
}

func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return 1
	}
	return exitErr.ExitCode()
}

func negateExitCode(code int, negated bool) int {
	if !negated {
		return code
	}
	if code == 0 {
		return 1
	}
	return 0
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
