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

// Behavior for the top-level is actually different
func Parse(conf string) (ConfLine, error) {
	lines := strings.Split(conf, "\n")
	if len(lines) == 0 {
		return ConfLine{}, nil
	}

	topLevel, _, err := parseSection(lines, indentLevel(lines[0]))
	if err != nil {
		return ConfLine{}, err
	}
	return ConfLine{"", topLevel}, nil
}

func parseSection(lines []string, minIndent int) ([]ConfLine, int, error) {
	var result []ConfLine

	idx := 0
	for idx < len(lines) {
		line := lines[idx]
		if strings.TrimSpace(line) == "" {
			idx++
			continue
		}
		indent := indentLevel(line)
		if indent < minIndent {
			return result, idx, nil
		}
		if indent == minIndent {
			result = append(result, ConfLine{strings.TrimSpace(line), nil})
			idx += 1
		}
		if indent > minIndent {
			section, skip, err := parseSection(lines[idx:], indent)
			if err != nil {
				return nil, 0, err
			}
			if skip == 0 {
				return nil, 0, fmt.Errorf("failed to advance in subsection at line %q", line)
			}
			idx += skip
			result[len(result)-1].SubLines = section
		}
	}
	return result, idx, nil
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
func Apply(config string, apply string) (string, error) {
	c, err := Parse(config)
	if err != nil {
		return "", err
	}
	a, err := Parse(apply)
	if err != nil {
		return "", err
	}

	result := c.Apply(&a)
	return result.String(), nil
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
