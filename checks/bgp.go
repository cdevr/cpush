package checks

import (
	"fmt"
	"strconv"

	"github.com/cdevr/cpush/textfsm"
)

func CheckBgpSum(router string, cmdResults map[string]string) ([]CheckResult, error) {
	var results []CheckResult
	checkName := "CheckBgpSum"

	if _, ok := cmdResults["show bgp sum"]; !ok {
		return []CheckResult{{checkName, router, "failed to get 'show bgp sum' command output"}}, nil
	}

	bgpSum, err := textfsm.ParseTypedCiscoIosShowBgpSummary(cmdResults["show bgp sum"])
	if err != nil {
		// Wrapping the error for more context might be good, but sticking to original structure for now.
		return nil, fmt.Errorf("couldnt parse show bgp sum result")
	}

	for _, neighbor := range bgpSum {
		// If neighbor.Status is an empty string, it implies textfsm parsed a numeric prefix count
		// and stored it elsewhere, leaving Status blank. This is a "good" state.
		if neighbor.Status == "" {
			continue
		}
		// If neighbor.Status can be converted to an integer, it's a prefix count (e.g., "0", "1", "10").
		// This is also a "good" state.
		if _, err := strconv.Atoi(neighbor.Status); err == nil {
			continue
		}
		// Otherwise, Status contains a non-numeric string like "Idle", "Active", "Connect",
		// which indicates a problem.
		results = append(results, CheckResult{checkName, router, fmt.Sprintf("%s: idle status %q", neighbor.RemoteIp, neighbor.Status)})
	}
	return results, nil
}
