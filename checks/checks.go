package checks

// No imports needed if CheckResult, CheckData, Checks, GetCheckCommands, and Check don't directly use them.
// However, the original `Checks` variable declaration *does* refer to `CheckInterfaces` and `CheckBgpSum`
// which are now in other files but part of the same package. The Go compiler will resolve these.
// The primary functions `GetCheckCommands` and `Check` themselves don't seem to require external packages.

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
		CheckInterfaces, // Defined in interfaces.go
	},
	{
		"BGP sessions",
		[]string{"show bgp sum"},
		CheckBgpSum, // Defined in bgp.go
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
			return nil, err // Propagate error
		}
		result = append(result, checkResults...)
	}
	return result, nil
}
