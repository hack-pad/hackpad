//go:build js
// +build js

package taskconsole

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/cmd/editor/ide"
)

type builder struct{}

func New() ide.TaskConsoleBuilder {
	return &builder{}
}

type console struct {
	stdout, stderr, note io.Writer
	commands             chan *exec.Cmd
	cancelFuncs          chan CancelFunc
	titleChan            chan string
}

func (b *builder) New(elem *dom.Element) ide.TaskConsole {
	elem.SetInnerHTML(`
<pre class="console-output"></pre>
`)
	elem.AddClass("console")
	outputElem := elem.QuerySelector(".console-output")
	c := &console{
		stdout:      newElementWriter(outputElem, ""),
		stderr:      newElementWriter(outputElem, "stderr"),
		note:        newElementWriter(outputElem, "note"),
		commands:    make(chan *exec.Cmd, 10),
		cancelFuncs: make(chan CancelFunc, 10),
		titleChan:   make(chan string, 1),
	}
	go c.runLoop()
	return c
}

func (c *console) Stdout() io.Writer { return c.stdout }
func (c *console) Stderr() io.Writer { return c.stderr }
func (c *console) Note() io.Writer   { return c.note }

func (c *console) Start(rawName, name string, args ...string) (context.Context, error) {
	if rawName == "" {
		rawName = name
	}
	cmd := exec.Command(name, args...)
	cmd.Path = rawName
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	c.commands <- cmd
	ctx, cancel := newCommandContext()
	c.cancelFuncs <- cancel
	return ctx, nil
}

func (c *console) runLoop() {
	c.titleChan <- "Build"
	for {
		c.runLoopIter()
	}
}

func (c *console) runLoopIter() {
	cmd := <-c.commands
	cancel := <-c.cancelFuncs
	startTime := time.Now()
	commandErr := c.runCmd(cmd)
	defer cancel(commandErr)
	elapsed := time.Since(startTime)
	if commandErr != nil {
		_, _ = c.stderr.Write([]byte(commandErr.Error()))
	}

	exitCode := 0
	if commandErr != nil {
		exitCode = 1
		exitCoder, ok := commandErr.(interface{ ExitCode() int })
		if ok {
			exitCode = exitCoder.ExitCode()
		}
	}

	_, _ = io.WriteString(c.Note(), fmt.Sprintf("%s (%.2fs)\n",
		exitStatus(exitCode),
		elapsed.Seconds(),
	))
}

func (c *console) runCmd(cmd *exec.Cmd) error {
	_, err := c.stdout.Write([]byte(fmt.Sprintf("$ %s\n", strings.Join(cmd.Args, " "))))
	if err != nil {
		return err
	}
	return cmd.Run()
}

func exitStatus(exitCode int) string {
	if exitCode == 0 {
		return "✔"
	}
	return "✘"
}

func (c *console) Titles() <-chan string {
	return c.titleChan
}
