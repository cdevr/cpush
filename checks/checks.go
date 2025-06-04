package checks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cdevr/cpush/textfsm"
)

type CheckResult struct {
	CheckName string
	Device    string
	Result    string
}

type CheckData struct {
	Name     string
	Commands []string
	F        func(router string, cmdResults map[string]string) ([]CheckResult, error)
}

var Checks = []CheckData{
	{
		"Interfaces",
		[]string{"show interfaces"},
		CheckInterfaces,
	},
	{
		"BGP sessions",
		[]string{"show bgp sum"},
		CheckBgpSum,
	},
	// TODO:
	// Check bootvar is set to 0x2102
	// Check BFD
	// Check interface transceiver
	// Check license all
	// Check standby HSRP
	{
		"Bootvar",
		[]string{"show version"},
		CheckBootvar,
	},
}

func GetCheckCommands() []string {
	var result []string
	for _, check := range Checks {
		result = append(result, check.Commands...)
	}
	return result
}

func Check(device string, cmdResults map[string]string) ([]CheckResult, error) {
	var result []CheckResult
	for _, c := range Checks {
		checkResults, err := c.F(device, cmdResults)
		if err != nil {
			return nil, err
		}
		result = append(result, checkResults...)
	}
	return result, nil
}

func CheckInterfaces(router string, cmdResults map[string]string) ([]CheckResult, error) {
	var results []CheckResult

	checkName := "CheckInterfaces"

	if _, ok := cmdResults["show interfaces"]; !ok {
		return []CheckResult{{checkName, router, "failed to get 'show interfaces' command output"}}, nil
	}

	interfaceResults, err := textfsm.ParseTypedCiscoIosShowInterfaces(cmdResults["show interfaces"])
	if err != nil {
		return nil, fmt.Errorf("couldnt parse interfaces result")
	}

	for _, ir := range interfaceResults {
		switch {
		// ok cases
		case ir.LinkStatus == "up" && ir.ProtocolStatus == "up":
		case ir.LinkStatus == "administratively down" && ir.ProtocolStatus == "down":
		default:
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: admin %q protocol %q", ir.Intf, ir.LinkStatus, ir.ProtocolStatus)})
		}

		if ir.Runts != "" && ir.Runts != "0" {
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: %s runts", ir.Intf, ir.Runts)})
		}
		if ir.Giants != "" && ir.Giants != "0" {
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: %s giants", ir.Intf, ir.Giants)})
		}
		if ir.InputErrors != "" && ir.InputErrors != "0" {
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: %s input errors", ir.Intf, ir.InputErrors)})
		}
		if ir.Crc != "" && ir.Crc != "0" {
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: %s CRC errors", ir.Intf, ir.Crc)})
		}
		if ir.Overrun != "" && ir.Overrun != "0" {
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: %s frame overruns", ir.Intf, ir.Overrun)})
		}
		if ir.Abort != "" && ir.Abort != "0" {
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: %s abort errors", ir.Intf, ir.Abort)})
		}
		if ir.OutputErrors != "" && ir.OutputErrors != "0" {
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: %s output errors", ir.Intf, ir.OutputErrors)})
		}
	}
	return results, nil
}

func CheckBootvar(router string, cmdResults map[string]string) ([]CheckResult, error) {
	var results []CheckResult
	checkName := "CheckBootvar"

	showVersionOutput, ok := cmdResults["show version"]
	if !ok {
		results = append(results, CheckResult{checkName, router, "failed to get 'show version' command output"})
		return results, nil
	}

	found := false
	for _, line := range strings.Split(showVersionOutput, "\n") {
		if strings.Contains(strings.ToLower(line), strings.ToLower("Configuration register is")) {
			found = true
			parts := strings.Fields(line)
			if len(parts) > 0 {
				actualValue := parts[len(parts)-1]
				if actualValue != "0x2102" {
					results = append(results, CheckResult{checkName, router, fmt.Sprintf("Configuration register is %s, expected 0x2102", actualValue)})
				}
			}
			break // Found the line, no need to check further
		}
	}

	if !found {
		results = append(results, CheckResult{checkName, router, "Configuration register line not found in 'show version' output"})
	}

	return results, nil
}

func CheckBgpSum(router string, cmdResults map[string]string) ([]CheckResult, error) {
	var results []CheckResult

	checkName := "CheckBgpSum"

	if _, ok := cmdResults["show bgp sum"]; !ok {
		return []CheckResult{{checkName, router, "failed to get 'show bgp sum' command output"}}, nil
	}

	bgpSum, err := textfsm.ParseTypedCiscoIosShowBgpSummary(cmdResults["show bgp sum"])
	if err != nil {
		return nil, fmt.Errorf("couldnt parse show bgp sum result")
	}

	for _, neighbor := range bgpSum {
		// Neighbor status should be the number of prefixes received. If it's anything else ("Idle", or "Connect", or "Active"), that's bad.
		if _, err := strconv.Atoi(neighbor.Status); err == nil {
			results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: idle status %q", neighbor.RemoteIp, neighbor.Status)})
		}
	}

	return results, nil
}
