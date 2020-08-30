package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/johnstarich/go-wasm/internal/console"
)

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
		return os.Chdir(dir)
	case 1:
		return os.Chdir(args[0])
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
