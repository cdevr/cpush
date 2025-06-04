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
