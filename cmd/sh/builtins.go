package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

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
