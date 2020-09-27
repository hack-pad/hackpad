package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/syntax"
)

func runCallExpr(term console.Console, stmt *syntax.Stmt, node *syntax.CallExpr, isPipe bool) error {
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
	cmd.Stdin = getConsoleStdin(term)
	cmd.Stdout = term.Stdout()
	cmd.Stderr = term.Stderr()

	for _, redir := range stmt.Redirs {
		err := applyRedirection(cmd, redir)
		if err != nil {
			return err
		}
	}

	err := runCmd(cmd, cmdOptions{
		Background: stmt.Background,
		Pipe:       isPipe,
	})
	err = exitErrFromCmd(err, stmt.Negated)
	return err

}

func applyRedirection(cmd *exec.Cmd, redir *syntax.Redirect) error {
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
	return nil
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
