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

func (c *ConfLine) IsLeaf() bool {
	if c == nil {
		return true
	}
	return len(c.SubLines) == 0
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

func (c *ConfLine) stringPrefix(prefix string) string {
	if c == nil {
		return ""
	}
	var result string
	if c.Line != "" {
		result += fmt.Sprintf("%s%s\n", prefix, c.Line)
	}
	for _, sl := range c.SubLines {
		newPrefix := fmt.Sprintf(" %s", prefix)
		if c.Line == "" {
			newPrefix = ""
		}
		result += sl.stringPrefix(newPrefix)
	}

	return result
}

func (c *ConfLine) String() string {
	return strings.TrimSuffix(c.stringPrefix(""), "\n")
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

	c.Apply(&a)
	return c.String(), nil
}

func (c *ConfLine) Apply(other *ConfLine) {
	if c.Line == other.Line {
		// process sublines
	outer:
		for _, osl := range other.SubLines {
			// TODO handle "no" prefix.
			// TODO needs special case handling, like "ip address" having a space but should still be considered one word.
			oslFirstWord := strings.Split(osl.Line, " ")[0]
			if osl.IsLeaf() {
				for cIndex, csl := range c.SubLines {
					cslFirstWord := strings.Split(csl.Line, " ")[0]
					if oslFirstWord == cslFirstWord && csl.IsLeaf() {
						c.SubLines[cIndex].Line = osl.Line
						continue outer
					}
				}
			} else {
				for _, csl := range c.SubLines {
					if osl.Line == csl.Line {
						csl.Apply(&osl)
					}
				}
			}
		}
	}
}
