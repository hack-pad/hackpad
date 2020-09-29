package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fatih/color"
)

const (
	promptTemplateStr = `{{.RCArrow}} {{.CurDirName}} $ `
)

var (
	promptTemplate = template.Must(template.New("").Parse(promptTemplateStr))
)

func prompt(term *terminal) string {
	s, err := promptErr(term)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to render prompt: ", err)
	}
	return s
}

func promptErr(term *terminal) (string, error) {
	var buf bytes.Buffer
	data, err := newPromptData(term)
	if err != nil {
		return "", err
	}
	err = promptTemplate.Execute(&buf, data)
	return buf.String(), err
}

type promptData struct {
	RCArrow    string
	CurDirName string
}

func newPromptData(term *terminal) (data *promptData, err error) {
	const rcArrow = "➜"
	data = &promptData{
		RCArrow: color.GreenString(rcArrow),
	}

	wd, err := os.Getwd()
	if err != nil {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	data.CurDirName = filepath.Base(wd)
	if wd == home {
		data.CurDirName = "~"
	}

	if term.lastExitCode != 0 {
		data.RCArrow = color.RedString(rcArrow)
	}

	return
}
