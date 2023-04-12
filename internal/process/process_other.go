//go:build !js
// +build !js

package process

import (
	"fmt"
	"os"
	"os/exec"
)

func (p *process) run(path string) {
	cmd := exec.Command(path, p.args...)
	if p.attr.Env == nil {
		cmd.Env = os.Environ()
		p.attr.Env = splitEnvPairs(cmd.Env)
	} else {
		for k, v := range p.attr.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	p.state = stateRunning
	prev := switchContext(p.pid)
	err := cmd.Run()
	switchContext(prev)
	p.exitCode = cmd.ProcessState.ExitCode()
	p.handleErr(err)
}
