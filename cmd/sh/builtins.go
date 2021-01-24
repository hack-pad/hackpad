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
	"time"

	"github.com/fatih/color"
	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/johnstarich/go/datasize"
	"github.com/pkg/errors"
)

type builtinFunc func(term console.Console, args ...string) error

var (
	builtins = map[string]builtinFunc{}
)

func init() {
	for k, v := range map[string]builtinFunc{
		"cat":   cat,
		"cd":    cd,
		"chmod": chmod,
		"clear": clear,
		"echo":  echo,
		"env":   env,
		"exit":  exit,
		"ls":    ls,
		"mkdir": mkdir,
		"mv":    mv,
		"pwd":   pwd,
		"rm":    rm,
		"rmdir": rmdir,
		"touch": touch,
		"which": which,
	} {
		builtins[k] = v
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
	set := flag.NewFlagSet("ls", flag.ContinueOnError)
	longForm := set.Bool("l", false, "Long format")
	err := set.Parse(args)
	if err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	args = set.Args()
	if len(args) == 0 {
		args = []string{"."}
	}
	if len(args) == 1 {
		return printFileNames(term, args[0], *longForm)
	}
	for _, f := range args {
		fmt.Fprintln(term.Stdout(), f+":")
		err := printFileNames(term, f, *longForm)
		if err != nil {
			return err
		}
		fmt.Fprintln(term.Stdout())
	}
	return nil
}

func printFileNames(term console.Console, dir string, longForm bool) error {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	if !longForm {
		for _, info := range infos {
			fmt.Fprintln(term.Stdout(), info.Name())
		}
		return nil
	}

	var t table
	t.Align(leftAlign, rightAlign)
	for _, info := range infos {
		value, units := formatBytes(datasize.Bytes(info.Size()))
		t.Add(info.Mode(), value, units, info.ModTime().Format(time.Stamp), info.Name())
	}
	fmt.Fprint(term.Stdout(), t)
	return nil
}

func formatBytes(b datasize.Size) (string, string) {
	value, unit := b.FormatSI()
	return strings.TrimSuffix(fmt.Sprintf("%.1f", value), ".0"), unit
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
		info, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.Errorf("Not a directory: %s", dir)
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
	if len(args) == 0 {
		_, err := io.Copy(term.Stdout(), getConsoleStdin(term))
		return err
	}

	for _, path := range args {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return errors.Errorf("%s: Is a directory", path)
		}
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
	cmd := exec.Command(args[0], args[1:]...) // nolint:gosec // Running any given process args is the whole point, so this isn't a security issue.
	cmd.Stdout = term.Stdout()
	cmd.Stderr = term.Stderr()
	cmd.Env = append(os.Environ(), env...)
	return runCmd(cmd, cmdOptions{})
}

func chmod(term console.Console, args ...string) error {
	if len(args) < 2 {
		return errors.New("Not enough args")
	}

	perm, err := strconv.ParseInt(args[0], 8, 12) // parse octal permission
	if err != nil {
		return err
	}
	file := args[1]
	return os.Chmod(file, os.FileMode(perm))
}
