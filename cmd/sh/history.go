package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var historyFile string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	historyFile = filepath.Join(home, ".history")
}

type history struct {
	lastIndex int
	lines     []string
}

func newHistory() (*history, error) {
	lines, err := loadHistoryFile()
	return &history{
		lines: lines,
	}, err
}

func loadHistoryFile() ([]string, error) {
	historyFile, err := os.Open(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, err
	}
	defer historyFile.Close()
	scanner := bufio.NewScanner(historyFile)
	if err != nil {
		return nil, err
	}
	var history []string
	for i := 0; i < 100 && scanner.Scan(); i++ {
		history = append(history, scanner.Text())
	}
	return history, nil
}

func (h *history) Push(command string) error {
	command = strings.TrimSpace(command)
	h.lastIndex = 0
	if command == "" || command == h.mostRecentCommand() {
		return nil
	}

	h.lines = append(h.lines, command)
	f, err := os.OpenFile(historyFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(command + "\n")
	return err
}

func (h *history) mostRecentCommand() string {
	if len(h.lines) > 0 {
		return h.lines[len(h.lines)-1]
	}
	return ""
}

func (h *history) Previous() (command string, ok bool) {
	if h.lastIndex < len(h.lines) {
		h.lastIndex++
		return h.lines[len(h.lines)-h.lastIndex], true
	}
	return "", false
}

func (h *history) Next() (command string, ok bool) {
	if h.lastIndex > 1 {
		h.lastIndex--
		return h.lines[len(h.lines)-h.lastIndex], true
	}
	ok = h.lastIndex == 1
	h.lastIndex = 0
	return "", ok
}
