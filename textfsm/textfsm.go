// Package textfsm Implementation of Google textfsm.
package textfsm

import (
	"bufio"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

type State struct {
	name  string
	rules []Rule
	fsm   *TextFSM
}

type Rule struct {
	Regex    string
	Match    string
	LineOp   string
	RecordOp string
	NewState string
	LineNum  int
}

var LineOperators = []string{"Continue", "Next", "Error"}
var RecordOperators = []string{"Clear", "Clearall", "Record", "NoRecord"}

func (r *Rule) String() string {
	var sb strings.Builder
	sb.WriteString(" " + r.Match)
	if r.LineOp != "" {
		sb.WriteString(" -> " + r.LineOp)
		if r.RecordOp != "" {
			sb.WriteString("." + r.RecordOp)
		}
		if r.NewState != "" {
			sb.WriteString(" " + r.NewState)
		}
	} else {
		if r.RecordOp != "" {
			sb.WriteString(" -> " + r.RecordOp)
			if r.NewState != "" {
				sb.WriteString(" " + r.NewState)
			}
		} else if r.NewState != "" {
			sb.WriteString(" -> " + r.NewState)
		}
	}
	return sb.String()
}
func (r *Rule) Parse(line string, lineNum int, varMap map[string]interface{}) error {
	r.LineNum = lineNum
	// Implicit default is '(regexp) -> Next.NoRecord'
	MatchAction := regexp.MustCompile(`(?P<match>.*)(\s->(?P<action>.*))`)
	// Line operators.
	OperRe := regexp.MustCompile(`(?P<ln_op>Continue|Next|Error)`)
	// Record operators.
	RecordRe := regexp.MustCompile(`(?P<rec_op>Clear|Clearall|Record|NoRecord)`)
	// Line operator with optional record operator.
	OperRecordRe := regexp.MustCompile(fmt.Sprintf("(%s(%s%s)?)", OperRe, `\.`, RecordRe))
	// New State or 'Error' string.
	NewstateRe := regexp.MustCompile(`(?P<new_state>\w+|".*")`)
	// Compound operator (line and record) with optional new state.
	ActionRe := regexp.MustCompile(fmt.Sprintf("^%s%s(%s%s)?$", `\s+`, OperRecordRe, `\s+`, NewstateRe))
	// Record operator with optional new state.
	Action2Re := regexp.MustCompile(fmt.Sprintf("^%s%s(%s%s)?$", `\s+`, RecordRe, `\s+`, NewstateRe))
	// Default operators with optional new state.
	Action3Re := regexp.MustCompile(fmt.Sprintf("^(%s%s)?$", `\s+`, NewstateRe))
	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("null data in Rule. Line: %d", r.LineNum)
	}
	// Is there '->' action present. ?
	matches := GetNamedMatches(MatchAction, line)
	if matches != nil {
		r.Match = matches["match"]
	} else {
		r.Match = line
	}
	if varMap != nil {
		regex, err := ExecutePythonTemplate(r.Match, varMap)
		if err != nil {
			return err
		}
		r.Regex = regex
	}
	if _, err := regexp.Compile(r.Regex); err != nil {
		return fmt.Errorf("line %d: Invalid regular expression '%s'. Error: '%s'", r.LineNum, r.Regex, err.Error())
	}
	if _, err := regexp.Compile(r.Match); err != nil {
		return fmt.Errorf("line %d: Invalid regular expression '%s'. Error: '%s'", r.LineNum, r.Match, err.Error())
	}
	action := matches["action"]
	m := GetNamedMatches(ActionRe, action)
	if m == nil {
		m = GetNamedMatches(Action2Re, action)
	}
	if m == nil {
		m = GetNamedMatches(Action3Re, action)
	}
	if m == nil {
		return fmt.Errorf("badly formatted rule '%s'. Line: %d", line, r.LineNum)
	}
	if lnOp, exists := m["ln_op"]; exists {
		r.LineOp = lnOp
	}
	if recOp, exists := m["rec_op"]; exists {
		r.RecordOp = recOp
	}
	if newState, exists := m["new_state"]; exists {
		r.NewState = newState
	}
	// Only 'Next' (or implicit 'Next') line operator can have a new_state.
	// But we allow error to have one as a warning message, so we are left
	// checking that Continue does not.
	if r.LineOp == "Continue" && r.NewState != "" {
		return fmt.Errorf("action '%s' with new state %s specified. Line: %d", r.LineOp, r.NewState, r.LineNum)
	}
	// Check that an error message is present only with the 'Error' operator.
	if r.LineOp != "Error" && r.NewState != "" {
		if !regexp.MustCompile(`^\w+$`).MatchString(r.NewState) {
			return fmt.Errorf("alphanumeric characters only in state names. Line: %d", r.LineNum)
		}
	}
	return nil
}

type TextFSM struct {
	CommentRe       *regexp.Regexp
	StateRe         *regexp.Regexp
	MaxStateNameLen int
	Values          map[string]Value
	States          map[string]State
	lineNum         int
}

// ParseString parses a string into a TextFSM structure.
func (t *TextFSM) ParseString(input string) error {
	return t.ParseReader(strings.NewReader(input))
}

func (t *TextFSM) ParseReader(reader *strings.Reader) error {
	return t.ParseScanner(bufio.NewScanner(reader))
}

func (t *TextFSM) ParseScanner(scanner *bufio.Scanner) error {
	t.CommentRe = regexp.MustCompile(`^\s*#`)
	t.StateRe = regexp.MustCompile(`^(\w+)$`)
	t.MaxStateNameLen = 48
	t.lineNum = 0
	err := t.parseFSMVariables(scanner)
	if err != nil {
		return err
	}
	t.States = make(map[string]State)
	for {
		done, err := t.parseFSMStates(scanner)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}
	err = t.validateFSM()
	if err != nil {
		return err
	}
	return nil
}

// Extracts Variables from start of template file.
//
//	    Values are expected as a contiguous block at the head of the file.
//	    These will be line separated from the State definitions that follow.
//	    Args:
//	      scanner: Scanner to read through lines
//		   Returns:
//	      returns error if there is any error while parsing. nil otherwise.
func (t *TextFSM) parseFSMVariables(scanner *bufio.Scanner) error {
	t.Values = make(map[string]Value)
	t.lineNum = 0
	for {
		t.lineNum++
		linePresent := scanner.Scan()
		if !linePresent {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("%d Line: Scanner Error %s", t.lineNum, err)
			}
			if t.lineNum == 1 {
				return fmt.Errorf("null template")
			}
			return fmt.Errorf("no State definition found")
		}
		line := scanner.Text()
		line = TrimRightSpace(line)
		// Blank line signifies end of Value definitions.
		if line == "" {
			return nil
		}
		// Skip commented lines.
		if t.CommentRe.MatchString(line) {
			continue
		}
		if strings.HasPrefix(line, "Value ") {
			value := Value{}
			err := value.Parse(line, t.lineNum)
			if err != nil {
				return err
			}
			t.Values[value.Name] = value
		} else if len(t.Values) == 0 {
			return fmt.Errorf("no Value definitions found")
		} else {
			return fmt.Errorf("expected blank line after last Value entry. Line: %d", t.lineNum)
		}
	}
}

// parseFSMStates extracts State and associated Rules from body of template file.
// After the Value definitions the remainder of the template is
// state definitions. The routine is expected to be called iteratively
// until no more states remain - indicated by returning None.
// The routine checks that the state names are a well-formed string, do
// not clash with reserved names and are unique.
// Args:
//
//	scanner: Scanner to read lines from
//
// Returns:
//
//			done bool: true if there are no lines left in the file. false if there are lines left to parse
//	     Error if there is any error. nil otherwise
func (t *TextFSM) parseFSMStates(scanner *bufio.Scanner) (done bool, err error) {
	for {
		t.lineNum++
		linePresent := scanner.Scan()
		if !linePresent {
			if err := scanner.Err(); err != nil {
				return true, fmt.Errorf("%d Line: Scanner Error %s", t.lineNum, err)
			}
			if len(t.States) == 0 {
				return true, fmt.Errorf("no State definition found")
			}
			return true, nil
		}
		line := scanner.Text()
		line = TrimRightSpace(line)
		if line == "" || t.CommentRe.MatchString(line) {
			continue
		}
		// First line is state definition
		if !t.StateRe.MatchString(line) {
			return false, fmt.Errorf("%d Line: Invalid state name '%s'", t.lineNum, line)
		}
		if len(line) > t.MaxStateNameLen {
			return false, fmt.Errorf("%d Line: state name too long. Should be < %d chars", t.lineNum, len(line))
		}
		if FindIndex(LineOperators, line) >= 0 || FindIndex(RecordOperators, line) >= 0 {
			return false, fmt.Errorf("%d Line: state '%s' can not be a keyword", t.lineNum, line)
		}
		if _, exists := t.States[line]; exists {
			return false, fmt.Errorf("%d Line: Duplicate state name '%s'", t.lineNum, line)
		}
		state := State{name: line, fsm: t}
		done, err = state.parseFSMRules(scanner)
		if err == nil {
			state.fsm.States[line] = state
		}
		return done, err
	}
}
func (t *State) parseFSMRules(scanner *bufio.Scanner) (done bool, err error) {
	t.rules = make([]Rule, 0)
	for {
		t.fsm.lineNum++
		linePresent := scanner.Scan()
		if !linePresent {
			if err := scanner.Err(); err != nil {
				return true, fmt.Errorf("%d Line: Scanner Error %s", t.fsm.lineNum, err)
			}
			// Looks like a state with no rules is fine?
			// if len(t.rules) == 0 {
			// 	return true, fmt.Errorf("No Rule definition found")
			// }
			return true, nil
		}
		line := scanner.Text()
		line = TrimRightSpace(line)
		// Empty line indicates the end of state
		if line == "" {
			return false, nil
		}
		if t.fsm.CommentRe.MatchString(line) {
			continue
		}
		valid := false
		for _, prefix := range []string{" ^", "  ^", "\t^"} {
			if strings.HasPrefix(line, prefix) {
				valid = true
				break
			}
		}
		if !valid {
			return false, fmt.Errorf("%d Line: Missing white space or carat ('^') before rule", t.fsm.lineNum)
		}
		rule := Rule{}
		varmap := make(map[string]interface{})
		for key, val := range t.fsm.Values {
			varmap[key] = val.Template
		}
		err = rule.Parse(line, t.fsm.lineNum, varmap)
		if err != nil {
			return false, err
		}
		t.rules = append(t.rules, rule)
	}
}

// Checks state names and destinations for validity.
// Each destination state must exist, be a valid name and
// not be a reserved name.
// There must be a 'Start' state and if 'EOF' or 'End' states are specified,
// they must be empty.
// Returns:
//
//	error if the FSM is invalid
func (t *TextFSM) validateFSM() error {
	// Must have 'Start' state.
	if _, exists := t.States["Start"]; !exists {
		return fmt.Errorf("missing state 'Start'")
	}
	// 'End/EOF' state (if specified) must be empty.
	if state, exists := t.States["End"]; exists {
		if state.rules != nil && len(state.rules) > 0 {
			return fmt.Errorf("non-Empty 'End' state")
		} else {
			// Remove 'End' state.
			delete(t.States, "End")
		}
	}
	if state, exists := t.States["EOF"]; exists {
		if state.rules != nil && len(state.rules) > 0 {
			return fmt.Errorf("non-Empty 'EOF' state")
		}
	}
	// Ensure jump states are all valid.
	for name, state := range t.States {
		for _, rule := range state.rules {
			if rule.LineOp == "Error" {
				continue
			}
			if rule.NewState == "" || rule.NewState == "End" || rule.NewState == "EOF" {
				continue
			}
			if _, exists := t.States[rule.NewState]; !exists {
				return fmt.Errorf("state '%s' not found, referenced in state '%s'", rule.NewState, name)
			}
		}
	}
	return nil
}

func TrimRightSpace(str string) string {
	return strings.TrimRightFunc(str, func(r rune) bool { return unicode.IsSpace(r) })
}
func GetNamedMatches(r *regexp.Regexp, s string) map[string]string {
	match := r.FindStringSubmatch(s)
	if match == nil {
		return nil
	}
	subMatchMap := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}
	return subMatchMap
}

// GetGroupNames returns a list of the group names for a given regular
// expression with named groups.
// ex. (?P<name>\w+)\s+(?P<age>\d+)
// Return the names e.g. ["name", "age"]
func GetGroupNames(r string) ([]string, error) {
	r1 := regexp.MustCompile("\\(\\?P<([a-z]+)>")
	m := r1.FindAllStringSubmatch(r, -1)
	output := make([]string, 0)
	if m == nil || len(m) == 0 {
		return output, nil
	}
	for _, arr := range m {
		if FindIndex(output, arr[1]) >= 0 {
			return nil, fmt.Errorf("duplicate name '%s'", arr[1])
		}
		output = append(output, arr[1])
	}
	return output, nil
}
func FindIndex(arr []string, elem string) int {
	for i, v := range arr {
		if v == elem {
			return i
		}
	}
	return -1
}

const maxNameLen = 48

type OnRecordType int

const (
	ortSkipRecord OnRecordType = iota
	ortSkipValue
	ortContinue
)

type Value struct {
	Regex         string
	Template      string
	Name          string
	Options       []string
	curval        interface{}
	filldownValue interface{}
}

func isValidOption(str string) bool {
	switch str {
	case "Required", "Key", "List", "Filldown", "Fillup":
		return true
	}
	return false
}

func (v *Value) Parse(input string, lineNum int) error {
	tokens := strings.Fields(input)
	if len(tokens) < 3 {
		return fmt.Errorf("%d Line: Expect at least 3 tokens on line", lineNum)
	}
	v.Options = make([]string, 0)
	if !strings.HasPrefix(tokens[2], "(") {
		// Format: Value Options Name Regular Expression
		// ex: Value Filledown,Required interface (.*)
		options := tokens[1]
		for _, option := range strings.Split(options, ",") {
			if !isValidOption(option) {
				return fmt.Errorf("line %d: Invalid option %s", lineNum, option)
			}
			idx := FindIndex(v.Options, option)
			if idx >= 0 {
				return fmt.Errorf("%d Line: Duplicate option %s", lineNum, option)
			}
			v.Options = append(v.Options, option)
		}
		v.Name = tokens[2]
		v.Regex = strings.Join(tokens[3:], " ")
	} else {
		// Format: Value Name Regular Expression
		// ex: Value interface (.*)
		v.Name = tokens[1]
		v.Regex = strings.Join(tokens[2:], " ")
	}
	if len(v.Name) > maxNameLen {
		return fmt.Errorf("%d Line: Invalid Value name '%s' or name too long", lineNum, v.Name)
	}
	squareBrackets := regexp.MustCompile(`([^\\]?)\[[^]]*]`)
	regexWithoutBrackets := squareBrackets.ReplaceAllString(v.Regex, "$1")
	if !regexp.MustCompile(`^\(.*\)$`).MatchString(v.Regex) {
		return fmt.Errorf("%d Line: Value '%s' must be contained within a '()' pair", lineNum, v.Regex)
	}
	if strings.Count(regexWithoutBrackets, "(") != strings.Count(regexWithoutBrackets, ")") {
		return fmt.Errorf("%d Line: Value '%s' must be contained within a '()' pair", lineNum, v.Regex)
	}
	if _, err := regexp.Compile(v.Regex); err != nil {
		return fmt.Errorf("line %d: Invalid regular expression '%s'. Error: '%s'", lineNum, v.Regex, err.Error())
	}
	if _, err := GetGroupNames(v.Regex); err != nil {
		return fmt.Errorf("line %d: Invalid group names. Error: %s", lineNum, err.Error())
	}
	v.Template = regexp.MustCompile(`^\(`).ReplaceAllString(v.Regex, fmt.Sprintf("(?P<%s>", v.Name))
	return nil
}

// String() returns a string representation of the value
func (v *Value) String() string {
	var sb strings.Builder
	sb.WriteString("Value ")
	if v.Options != nil && len(v.Options) > 0 {
		sb.WriteString(strings.Join(v.Options, ","))
		sb.WriteString(" ")
	}
	sb.WriteString(fmt.Sprintf("%s %s", v.Name, v.Regex))
	return sb.String()
}

func (v *Value) processScalarValue(newval string) {
	var finalval interface{} = nil
	if FindIndex(v.Options, "List") >= 0 {
		// If the value is 'List', add the new value to the current value.
		if v.curval == nil {
			if FindIndex(v.Options, "Filldown") >= 0 && v.filldownValue != nil {
				// curval is null. But there is a filldown value. Append to filldown value
				finalval = append(v.filldownValue.([]string), newval)
			} else {
				finalval = make([]string, 0)
				finalval = append(finalval.([]string), newval)
			}
		} else {
			finalval = append(v.curval.([]string), newval)
		}
	} else {
		finalval = newval
	}
	if FindIndex(v.Options, "Filldown") >= 0 {
		// If there is Filldown present, Remember the new value as filldown value
		if finalval == nil {
			finalval = v.filldownValue
		} else {
			v.filldownValue = finalval
		}
	}
	v.curval = finalval
}

func (v *Value) processMapValue(newval map[string]string) {
	newMap := make(map[string]string)
	varNames, err := GetGroupNames(v.Regex)
	if err != nil {
		panic(err)
	}
	for _, name := range varNames {
		newMap[name] = newval[name]
	}
	var finalval interface{} = newMap
	if FindIndex(v.Options, "List") >= 0 {
		// If the value is 'List', add the new value to the current value.
		if newval != nil && len(newval) > 0 {
			if v.curval == nil {
				if FindIndex(v.Options, "Filldown") >= 0 && v.filldownValue != nil {
					// curval is null. But there is a filldown value. Append to filldown value
					finalval = append(v.filldownValue.([]map[string]string), newMap)
				} else {
					finalval = make([]map[string]string, 0)
					finalval = append(finalval.([]map[string]string), newMap)
				}
			} else {
				finalval = append(v.curval.([]map[string]string), newMap)
			}
		}
	}
	if FindIndex(v.Options, "Filldown") >= 0 {
		// If there is Filldown present, Remember the new value as filldown value
		if finalval == nil {
			finalval = v.filldownValue
		} else {
			v.filldownValue = finalval
		}
	}
	v.curval = finalval
}

func (v *Value) onAppendRecord() OnRecordType {
	if FindIndex(v.Options, "Required") >= 0 {
		if v.isEmptyValue(v.curval) {
			if FindIndex(v.Options, "Filldown") >= 0 {
				if v.isEmptyValue(v.filldownValue) {
					return ortSkipRecord
				} else {
					return ortContinue
				}
			}
			return ortSkipRecord
		}
	}
	return ortContinue
}

func (v *Value) clearValue(all bool) {
	v.curval = nil
	if all && FindIndex(v.Options, "Filldown") >= 0 {
		v.filldownValue = nil
	}
}

func (v *Value) getFinalValue() interface{} {
	if v.isEmptyValue(v.curval) && FindIndex(v.Options, "Filldown") >= 0 {
		return v.getFinalValueInternal(v.filldownValue)
	}
	return v.getFinalValueInternal(v.curval)
}
func (v *Value) getFinalValueInternal(val interface{}) interface{} {
	if val == nil {
		if idx := FindIndex(v.Options, "List"); idx >= 0 {
			if strings.Contains(v.Regex, "(?P") {
				// If the regex contains (?P
				// ex: Value List ((?P<name>\w+)\s+(?P<age>\d+))
				// This will be an array of maps.
				return make([]map[string]string, 0)
			}
			// Else, it will be an array of strings
			return make([]string, 0)
		} else if strings.Contains(v.Regex, "(?P") {
			return make(map[string]string)
		} else {
			return ""
		}
	}
	return val
}

func (v *Value) isEmptyValue(val interface{}) bool {
	if val == nil {
		return true
	}
	switch val.(type) {
	case string:
		return val.(string) == ""
	case []string:
		return len(val.([]string)) == 0
	case map[string]string:
		return len(val.(map[string]string)) == 0
	case []map[string]string:
		return len(val.([]map[string]string)) == 0
	default:
		panic(fmt.Sprintf("Unknown data type %v for %s", reflect.TypeOf(val), v.Name))
	}

}

// ParserOutput has the output state.
// Dict contains a slice of maps. Each element in the slice holds the value of a record.
// Each record is represented as map of (name,value)
//
// Note that type of value is interface{}. But the concrete type is either 'string' or '[]string'
type ParserOutput struct {
	Dict         []map[string]interface{}
	lineNum      int
	curStateName string
}

func (t *ParserOutput) Reset(fsm TextFSM) {
	t.clearRecord(fsm, true)
	t.curStateName = "Start"
	t.Dict = make([]map[string]interface{}, 0)
}

// ParseTextString passes CLI output (provided as string) through FSM and
//
//	    Args:
//	      text: (string), Text to parse with embedded newlines.
//			 fsm: (TextFSM), TextFSM object as a result of parsing the text fsm template
//	      eof: (bool), Set to False if we are parsing only part of the file.
//	            Suppresses triggering EOF state.
//	    Returns:
//	      error if there is any error in parsing
func (t *ParserOutput) ParseTextString(text string, fsm TextFSM, eof bool) error {
	return t.ParseTextReader(strings.NewReader(text), fsm, eof)
}

func (t *ParserOutput) ParseTextReader(reader *strings.Reader, fsm TextFSM, eof bool) error {
	return t.ParseTextScanner(bufio.NewScanner(reader), fsm, eof)
}

func (t *ParserOutput) ParseTextScanner(scanner *bufio.Scanner, fsm TextFSM, eof bool) error {
	t.lineNum = 0
	if t.curStateName == "" {
		t.curStateName = "Start"
	}
	if t.Dict == nil {
		t.Dict = make([]map[string]interface{}, 0)
	}
	for {
		t.lineNum++
		linePresent := scanner.Scan()
		if !linePresent {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("%d Line: Scanner Error %s", t.lineNum, err)
			}
			break
		}
		line := scanner.Text()
		err := t.checkLine(line, fsm)
		if err != nil {
			return err
		}
		if t.curStateName == "End" || t.curStateName == "EOF" {
			break
		}
	}
	_, eofExists := fsm.States["EOF"]
	if t.curStateName != "End" && (!eofExists) && eof {
		// Implicit EOF performs Next.Record operation.
		// Suppressed if Null EOF state is instantiated.
		t.appendRecord(fsm)
	}
	return nil
}

// checkLine passes the line through each rule until a match is made.
// If the value regex contains nested match groups in the form (?P<name>regex),
//
//	    In case of List type with nested match groups
//	       instead of adding a string to the list, we add a dictionary of the groups.
//	    Other value types with nested match groups,
//	        the value is set as 'map[string]string' instead of a 'string'
//	    E.g.
//	    Value List ((?P<name>\w+)\s+(?P<age>\d+)) would create results like:
//	        [{'name': 'Bob', 'age': 32}]
//	    Do not give nested groups the same name as other values in the template.
//	    Nested regexps more than 2 levels are not supported currently
//	    Args:
//	      line: A string, the current input line.
//			 fsm: TextFSM Object
func (t *ParserOutput) checkLine(line string, fsm TextFSM) error {
	// fmt.Printf("Looking at line '%s'\n", line)
	state, exists := fsm.States[t.curStateName]
	if !exists {
		// Should never happen for a proper TextFSM
		panic(fmt.Sprintf("Unknown State %s", t.curStateName))
	}
	for _, rule := range state.rules {
		varmap := GetNamedMatches(regexp.MustCompile(rule.Regex), line)
		if varmap != nil {
			// fmt.Printf("Line '%s'. Regex: '%s' varmap: '%v'\n", line, rule.Regex, varmap)
			for key, val := range varmap {
				valobj, exists := fsm.Values[key]
				if !exists {
					// This may happen in case of nested match groups.
					// There will be no TextFSMValue with the names inside the nested match groups.
					continue
				}
				if strings.Contains(valobj.Regex, "(?P") {
					valobj.processMapValue(varmap)
				} else {
					valobj.processScalarValue(val)
				}
				if FindIndex(valobj.Options, "Fillup") >= 0 && valobj.curval != nil && t.Dict != nil {
					for i := len(t.Dict) - 1; i >= 0; i-- {
						if valobj.isEmptyValue(t.Dict[i][key]) {
							t.Dict[i][key] = valobj.curval
						} else {
							break
						}
					}
				}
				// For some reason, modifying curval in valobj using processValue is not reflecting.
				// Setting it back to fsm.Values works. Need to understand this further
				fsm.Values[key] = valobj
			}
			output, err := t.handleOperations(rule, fsm, line)
			if err != nil {
				return err
			}
			if output {
				if rule.NewState != "" {
					t.curStateName = rule.NewState
				}
				break
			}
		}
	}
	// fmt.Printf("After Line: '%s: ' current state: '%s'\n", line, t.cur_state_name)

	// for name, varobj := range fsm.Values {
	// 	fmt.Printf(" %s: curval '%v', filldownval '%v', ", name, varobj.curval, varobj.filldown_value)
	// }
	// fmt.Printf("\n")
	return nil
}

// appendRecord adds current record to result if well-formed.
func (t *ParserOutput) appendRecord(fsm TextFSM) {
	newMap := make(map[string]interface{})
	anyValue := false
	for name, value := range fsm.Values {
		ret := value.onAppendRecord()
		switch ret {
		case ortSkipRecord:
			t.clearRecord(fsm, false)
			return
		case ortSkipValue:
			newMap[name] = nil
		case ortContinue:
			newMap[name] = value.getFinalValue()
			if !value.isEmptyValue(newMap[name]) {
				anyValue = true
			}
		}
	}
	// If no Values in template or whole record is empty then don't output.
	if anyValue {
		t.Dict = append(t.Dict, newMap)
	}
	t.clearRecord(fsm, false)
}

// handleOperation handles Operators on the data record.
//
// Operators come in two parts and are a '.' separated pair:
//
//	  Operators that effect the input line or the current state (line_op).
//		'Next'      Get next input line and restart parsing (default).
//		'Continue'  Keep current input line and continue resume parsing.
//		'Error'     Unrecoverable input discard result and raise Error.
//
//
//	  Operators that affect the record being built for output (record_op).
//		'NoRecord'  Does nothing (default)
//		'Record'    Adds the current record to the result.
//		'Clear'     Clears non-Filldown data from the record.
//		'Clearall'  Clears all data from the record.
//
// Args:
//
//	rule: FSMRule object.
//	line: A string, the current input line.
//
// Returns:
//
//	True if state machine should restart state with new line.
//	error: If Error state is encountered.
func (t *ParserOutput) handleOperations(rule Rule, fsm TextFSM, line string) (output bool, err error) {
	if rule.RecordOp == "Record" {
		t.appendRecord(fsm)
	}
	if rule.RecordOp == "Clear" {
		t.clearRecord(fsm, false)
	}
	if rule.RecordOp == "Clearall" {
		t.clearRecord(fsm, true)
	}
	if rule.LineOp == "Error" {
		if rule.NewState != "" {
			return false, fmt.Errorf("error: %s. Rule Line: %d. Input Line: %s", rule.NewState, rule.LineNum, line)
		} else {
			return false, fmt.Errorf("state Error raised. Rule Line: %d. Input Line: %s", rule.LineNum, line)
		}
	} else if rule.LineOp == "Continue" {
		return false, nil
	}
	return true, nil
}

func (t *ParserOutput) clearRecord(fsm TextFSM, all bool) {
	for name, value := range fsm.Values {
		value.clearValue(all)
		// For some reason, modifying curval in valobj using processValue is not reflecting.
		// Setting it back to fsm.Values works. Need to understand this further
		fsm.Values[name] = value
	}
}

// ExecutePythonTemplate converts a python template into a golang template and executes
// against the given variable map.
// This is a quick hack and not does not support all comprehensive list of Python Template package.
// It does the following:
//   - replace all ${varname} to {{.varname}}
//   - replace all $varname to {{.varname}} (By sorting the varnames with the longest name first)
//   - replace all $$ with $
//   - escape all literal {{ and }} with {{"{{"}} and {{"}}"}}
//
// Assumes it is a valid python Template. No validations done to validate the python template syntax.
// Assumes all the variables are proper variable identifiers
// Then executes the resulting golang template on the map passed.
func ExecutePythonTemplate(pytemplate string, varsMap map[string]interface{}) (string, error) {
	t := pytemplate
	// Replace ${xxxx} with {{.xxxx}}
	r1 := regexp.MustCompile(`\${([^${}]+)}`)
	t = r1.ReplaceAllString(t, "__DOUBLE_OPENBR__ .$1 __DOUBLE_CLOSEBR__")
	if varsMap != nil {
		keys := make([]string, 0, len(varsMap))
		for k := range varsMap {
			keys = append(keys, k)
		}
		// Sort by longest key first.
		// This is done so that $var1234 is replaced first before replacing $var12 for example.
		sort.SliceStable(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
		for _, key := range keys {
			t = regexp.MustCompile(`\$(`+key+`)`).ReplaceAllString(t, "__DOUBLE_OPENBR__ .$1 __DOUBLE_CLOSEBR__")
		}
	}
	// Replace $$ with $
	t = strings.ReplaceAll(t, "$$", "$")
	// Escape { and } with {{"{"}} and {{"}"}}
	// Generally speaking, golang template has special meeaning for {{ and }}. Hence, we should escape only {{ and }}
	//
	// But. Looks like golang template barks at something like \{{{.INBOUND_SETTINGS_IN_USE}}
	// Hence, we escape every single { and } instead of only {{ and }}
	t = strings.ReplaceAll(t, "{", `__DOUBLE_OPENBR__"{"__DOUBLE_CLOSEBR__`)
	t = strings.ReplaceAll(t, "}", `__DOUBLE_OPENBR__"}"__DOUBLE_CLOSEBR__`)
	t = strings.ReplaceAll(t, "__DOUBLE_OPENBR__", `{{`)
	t = strings.ReplaceAll(t, "__DOUBLE_CLOSEBR__", `}}`)
	var sb strings.Builder
	gotemplate, err := template.New("test").Parse(t)
	if err != nil {
		return "", err
	}
	err = gotemplate.Execute(&sb, varsMap)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}
