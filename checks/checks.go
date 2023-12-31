package checks

import (
	"fmt"
	"strconv"

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
