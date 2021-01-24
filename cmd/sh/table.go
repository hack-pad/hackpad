package main

import (
	"fmt"
	"strings"
)

type table struct {
	rows  [][]string
	align []columnAlign
}

type columnAlign int

const (
	leftAlign columnAlign = iota
	rightAlign
)

func (t *table) Align(alignment ...columnAlign) {
	t.align = alignment
}

func (t *table) Add(columns ...interface{}) {
	stringColumns := make([]string, len(columns))
	for i := range columns {
		stringColumns[i] = strings.TrimSpace(fmt.Sprint(columns[i]))
	}
	t.rows = append(t.rows, stringColumns)
}

func (t table) String() string {
	var columnWidths []int
	for _, row := range t.rows {
		if len(columnWidths) < len(row) {
			columnWidths = append(columnWidths, make([]int, len(row)-len(columnWidths))...)
		}
		for ix, col := range row {
			if len(col) > columnWidths[ix] {
				columnWidths[ix] = len(col)
			}
		}
	}

	const colSeparator = ' '
	var s strings.Builder
	for _, row := range t.rows {
		for ix, col := range row {
			align := leftAlign
			if len(t.align) > ix {
				align = t.align[ix]
			}
			switch align {
			case rightAlign:
				s.WriteString(padString(columnWidths[ix] - len(col)))
				s.WriteString(col)
			default:
				s.WriteString(col)
				s.WriteString(padString(columnWidths[ix] - len(col)))
			}
			s.WriteRune(colSeparator)
		}
		s.WriteRune('\n')
	}
	return s.String()
}

func padString(length int) string {
	return fmt.Sprintf(fmt.Sprintf("%%%ds", length), "")
}
