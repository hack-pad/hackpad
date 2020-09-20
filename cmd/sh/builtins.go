package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"syscall/js"

	"github.com/fatih/color"
	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/pkg/errors"
)

type builtinFunc func(term console.Console, args ...string) error

var (
	builtins   = map[string]builtinFunc{}
	jsFunction = js.Global().Get("Function")
	goWasm     = js.Global().Get("goWasm")
)

func init() {
	builtins = map[string]builtinFunc{
		"cat":    cat,
		"cd":     cd,
		"clear":  clear,
		"echo":   echo,
		"env":    env,
		"exit":   exit,
		"jseval": jseval,
		"ls":     ls,
		"mkdir":  mkdir,
		"mv":     mv,
		"pwd":    pwd,
		"rm":     rm,
		"rmdir":  rmdir,
		"touch":  touch,
		"which":  which,
		"wpk":    wpk,
	}
}

func echo(term console.Console, args ...string) error {
	fmt.Fprintln(term.Stdout(), strings.Join(args, " "))
	return nil
}

func pwd(term console.Console, args ...string) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Fprintln(term.Stdout(), path)
	return nil
}

func ls(term console.Console, args ...string) error {
	if len(args) == 0 {
		args = []string{"."}
	}
	if len(args) == 1 {
		return printFileNames(term, args[0])
	}
	for _, f := range args {
		fmt.Fprintln(term.Stdout(), f+":")
		err := printFileNames(term, f)
		if err != nil {
			return err
		}
		fmt.Fprintln(term.Stdout())
	}
	return nil
}

func printFileNames(term console.Console, dir string) error {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, info := range infos {
		fmt.Fprintln(term.Stdout(), info.Name())
	}
	return nil
}

func cd(term console.Console, args ...string) error {
	switch len(args) {
	case 0:
		dir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		args = []string{dir}
		fallthrough
	case 1:
		dir := args[0]
		if _, err := os.Stat(dir); err != nil {
			return err
		}
		return os.Chdir(dir)
	default:
		return errors.New("Too many args")
	}
}

func mkdir(term console.Console, args ...string) error {
	switch len(args) {
	case 0:
		return errors.New("Must provide a path to create a directory")
	default:
		for _, dir := range args {
			err := os.Mkdir(dir, 0755)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func cat(term console.Console, args ...string) error {
	for _, path := range args {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(term.Stdout(), f)
		if err != nil {
			return err
		}
	}
	return nil
}

func mv(term console.Console, args ...string) error {
	switch len(args) {
	case 0, 1:
		return errors.New("Not enough args")
	case 2:
		src := args[0]
		dest := args[1]
		if strings.HasSuffix(dest, "/") {
			dest += path.Base(src)
		}
		return os.Rename(src, dest)
	default:
		return errors.New("Too many args")
	}
}

func rm(term console.Console, args ...string) error {
	set := flag.NewFlagSet("rm", flag.ContinueOnError)
	recursive := set.Bool("r", false, "Remove recursively")
	if err := set.Parse(args); err != nil {
		return err
	}

	if set.NArg() == 0 {
		return errors.New("Not enough args")
	}

	rmFunc := os.RemoveAll
	if !*recursive {
		rmFunc = func(path string) error {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return &os.PathError{Path: path, Op: "remove", Err: syscall.EISDIR}
			}
			return os.Remove(path)
		}
	}
	for _, f := range set.Args() {
		err := rmFunc(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func rmdir(term console.Console, args ...string) error {
	if len(args) == 0 {
		return errors.New("Not enough args")
	}
	for _, path := range args {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return &os.PathError{Path: path, Op: "remove", Err: syscall.ENOTDIR}
		}
		err = os.Remove(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func touch(term console.Console, args ...string) error {
	if len(args) == 0 {
		return errors.New("Not enough args")
	}
	for _, path := range args {
		_, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if os.IsNotExist(err) {
			f, err := os.Create(path)
			if err != nil {
				return err
			}
			err = f.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func which(term console.Console, args ...string) error {
	if len(args) == 0 {
		return errors.New("Not enough args")
	}
	for _, arg := range args {
		path, err := exec.LookPath(arg)
		if err != nil {
			return err
		}
		fmt.Fprintln(term.Stdout(), path)
	}
	return nil
}

func clear(term console.Console, args ...string) error {
	term.(*terminal).Clear()
	return nil
}

func exit(term console.Console, args ...string) error {
	if len(args) == 0 {
		os.Exit(0)
	}

	exitCode, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}
	fmt.Fprintf(term.Stderr(), color.RedString("Exited with code %d\n"), exitCode)
	os.Exit(int(exitCode))
	return nil
}

func env(term console.Console, args ...string) error {
	var kv []string
	const equals = '='
	for i, arg := range args {
		if !strings.ContainsRune(arg, equals) {
			args = args[i:]
			break
		}
		kv = append(kv, arg)
	}

	if len(args) == 0 {
		for _, e := range os.Environ() {
			fmt.Fprintln(term.Stdout(), e)
		}
		return nil
	}

	return runWithEnv(term, kv, args...)
}

func splitKeyValue(kv string) (key, value string) {
	const equals = "="
	tokens := strings.SplitN(kv, equals, 2)
	if len(tokens) < 2 {
		return strings.Join(tokens, equals), ""
	}
	return tokens[0], strings.Join(tokens[1:], equals)
}

func runWithEnv(term console.Console, env []string, args ...string) error {
	builtin, ok := builtins[args[0]]
	if ok {
		var oldKV, unsetKV []string
		// override env for builtin
		for _, pair := range env {
			key, value := splitKeyValue(pair)
			if oldValue, isSet := os.LookupEnv(key); isSet {
				oldKV = append(oldKV, key+"="+oldValue)
			} else {
				unsetKV = append(unsetKV, key)
			}
			os.Setenv(key, value)
		}
		err := builtin(term, args[1:]...)
		// restore env
		for _, pair := range oldKV {
			key, value := splitKeyValue(pair)
			os.Setenv(key, value)
		}
		for _, key := range unsetKV {
			os.Unsetenv(key)
		}
		return err
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(), env...)
	return cmd.Run()
}

func jsEval(funcStr string, args ...interface{}) js.Value {
	f := jsFunction.Invoke(`"use strict";` + funcStr)
	return f.Invoke(args...)
}

func jseval(term console.Console, args ...string) error {
	if len(args) < 1 {
		return errors.New("Must provide a string to run as a function")
	}
	result := jsEval(args[0], strings.Join(args[1:], " "))
	fmt.Fprintln(term.Stdout(), result)
	return nil
}

func wpk(term console.Console, args ...string) error {
	if len(args) < 2 {
		return errors.New(strings.TrimSpace(`
Usage: wpk add <pkg>

Installs a remote package by the name of 'pkg'.
`))
	}
	switch args[0] {
	case "add":
		prom := promise.From(goWasm.Call("install", args[1]))
		_, err := promise.Await(prom)
		return err
	default:
		return errors.Errorf("Invalid command: %q", args[0])
	}
}
