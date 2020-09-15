package taskconsole

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/cmd/editor/ide"
)

type builder struct{}

func New() ide.TaskConsoleBuilder {
	return &builder{}
}

type console struct {
	stdout, stderr, note io.Writer
	commands             chan *exec.Cmd
	runningCommands      sync.WaitGroup
	titleChan            chan string
}

func (b *builder) New(element js.Value) ide.TaskConsole {
	element.Set("innerHTML", `
<pre class="console-output"></pre>
`)
	element.Get("classList").Call("add", "console")
	outputElem := element.Call("querySelector", ".console-output")
	c := &console{
		stdout:    newElementWriter(outputElem, ""),
		stderr:    newElementWriter(outputElem, "stderr"),
		note:      newElementWriter(outputElem, "note"),
		commands:  make(chan *exec.Cmd, 10),
		titleChan: make(chan string, 1),
	}
	go c.runLoop()
	return c
}

func (c *console) Stdout() io.Writer { return c.stdout }
func (c *console) Stderr() io.Writer { return c.stderr }
func (c *console) Note() io.Writer   { return c.note }

func (c *console) Start(rawName, name string, args ...string) error {
	if rawName == "" {
		rawName = name
	}
	cmd := exec.Command(name, args...)
	cmd.Path = rawName
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	c.commands <- cmd
	return nil
}

func (c *console) runLoop() {
	c.titleChan <- "Build"
	for {
		cmd := <-c.commands
		c.runningCommands.Add(1)
		startTime := time.Now()
		commandErr := c.runCmd(cmd)
		if commandErr != nil {
			_, _ = c.stderr.Write([]byte(commandErr.Error()))
		}
		elapsed := time.Since(startTime)
		c.runningCommands.Done()

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
}

func (c *console) runCmd(cmd *exec.Cmd) error {
	_, err := c.stdout.Write([]byte(fmt.Sprintf("$ %s\n", strings.Join(cmd.Args, " "))))
	if err != nil {
		return err
	}
	return cmd.Run()
}

func (c *console) Wait() error {
	c.runningCommands.Wait()
	return nil
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
