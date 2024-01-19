// Package diff contains functions to diff ios config files and to apply configlets in simulation.
package cisco

import (
	"fmt"
	"strings"
)

func indentLevel(s string) int {
	if strings.Trim(s, " ") == "" {
		return 0
	}
	result := 0
	for _, c := range s {
		if c == ' ' {
			result++
		} else {
			break
		}
	}
	return result
}

// ConfigToFormal puts normal cisco config in IOS XR "formal" config, somewhat like Juniper's "set" config.
//
// An example can explain this better:
//
//	interface loopback0
//	  description boembabies
//
// becomes:
//
//	interface loopback0
//	interface loopback0 description boembabies
func ConfigToFormal(c1 string) string {
	var result []string
	lines := strings.Split(c1, "\n")
	if len(lines) == 0 {
		return ""
	}
	lastHeadings := []string{lines[0]}
	lastIndent := 0
	for _, line := range lines {
		lineIndent := indentLevel(line)
		switch {
		case lineIndent > lastIndent:
			lastHeadings = append(lastHeadings, line)
			lastIndent = lineIndent
		case lineIndent < lastIndent:
			lastHeadings = lastHeadings[:len(lastHeadings)-1]
			lastIndent = lineIndent
			lastHeadings[lineIndent] = line
		default:
			lastHeadings[lineIndent] = line
		}
		result = append(result, strings.Join(lastHeadings[:lineIndent], " ")+line)
	}
	return strings.Join(result, "\n")
}

// A cisco configuration consists of "conflines", which are really just lines,
// that can start a section. For example:
//
// interface loopback0
//
//	description boembabies
type ConfLine struct {
	Line     string
	SubLines []ConfLine
}

func Parse(conf string) ConfLine {
	lines := strings.Split(conf, "\n")
	if len(lines) == 0 {
		return ConfLine{"", nil}
	}
	return parseLines(lines, true)
}

func parseLines(lines []string, top bool) ConfLine {
	if len(lines) == 0 {
		return ConfLine{"", nil}
	}

	result := ConfLine{lines[0], nil}
	if top {
		result = ConfLine{"", nil}
	}
	lastIndent := indentLevel(lines[0])

	for idx, line := range lines {
		if idx == 0 && top == false {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.TrimSpace(line) == "!" {
			continue
		}
		lineIndent := indentLevel(line)
		switch {
		case lineIndent > lastIndent:
			subLine := parseLines(lines[idx:], false)
			result.SubLines = append(result.SubLines, subLine)
		case lineIndent < lastIndent:
			return result
		default:
			if top {
				subLine := ConfLine{line, nil}
				result.SubLines = append(result.SubLines, subLine)
			} else {
				return result
			}
		}
	}
	return result
}

func (c *ConfLine) StringPrefix(prefix string) string {
	if c == nil {
		return ""
	}
	var result string
	if c.Line != "" {
		result += fmt.Sprintf("%s\n", c.Line)
	}
	for _, sl := range c.SubLines {
		newPrefix := fmt.Sprintf(" %s", prefix)
		if c.Line == "" {
			newPrefix = ""
		}
		result += sl.StringPrefix(newPrefix)
	}

	return result
}

func (c *ConfLine) String() string {
	return strings.TrimSuffix(c.StringPrefix(""), "\n")
}

// Apply applies a configlet to a router config and returns the result.
func Apply(config string, apply string) string {
	c := Parse(config)
	a := Parse(apply)

	result := c.Apply(&a)
	return result.String()
}

func (c *ConfLine) Apply(a *ConfLine) ConfLine {
	splitC := strings.Split(c.Line, " ")
	splitA := strings.Split(a.Line, " ")

	result := ConfLine{}
	// If this is not a section start, just replace.
	if splitC[0] == splitA[0] && len(c.SubLines) == len(a.SubLines) && len(c.SubLines) == 0 {
		result.Line = a.Line
		return result
	}
	// If this is a section start, dive into it if the first line matches entirely
	if len(c.SubLines) != 0 && len(a.SubLines) != 0 {

	}

	return result
}
