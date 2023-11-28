// Package diff contains functions to diff ios config files and to apply configlets in simulation.
package cisco

import "strings"

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
		default:
			lastHeadings[lineIndent] = line
		}
		result = append(result, strings.Join(lastHeadings[:lineIndent], " ")+line)
	}
	return strings.Join(result, "\n")
}

func Compare(c1, c2 string) (equal bool, diff string, err error) {
	return false, "", nil
}
