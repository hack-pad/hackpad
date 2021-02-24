package main

import "fmt"

type ExitErr struct {
	Code int
}

func (e *ExitErr) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}
