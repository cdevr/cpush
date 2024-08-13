package checks

import (
	"github.com/go-test/deep"
	"testing"
)

func TestCheckInterfaces(t *testing.T) {
	dev := "router1"

	tests := []struct {
		Comment    string
		Device     string
		CmdResults map[string]string
		WantErr    error
		Want       []CheckResult
	}{
		{
			"good interface",
			dev,
			map[string]string{"show interfaces": `GigabitEthernet0/1 is up, line protocol is up  
Description: good
`},
			nil, // no error
			nil,
		},
		{
			"good interface, but down",
			dev,
			map[string]string{"show interfaces": `GigabitEthernet0/1 is administratively down, line protocol is down 
Description: also good
`},
			nil, // no error
			nil,
		},
		{
			"wrong IntfStatus: line protocol shouldn't be down",
			dev,
			map[string]string{"show interfaces": `GigabitEthernet0/1 is up, line protocol is down 
Description: bad because admin up line down
`},
			nil, // no error
			[]CheckResult{{"CheckInterfaces", dev, "GigabitEthernet0/1: admin \"up\" protocol \"down\""}},
		},
		{
			"wrong IntfStatus: Input errors",
			dev,
			map[string]string{"show interfaces": `GigabitEthernet0/1 is up, line protocol is up 
Description: bad because input errors
      33 input errors, 0 CRC, 0 frame, 0 overrun, 0 ignored, 0 abort
`},
			nil, // no error
			[]CheckResult{{"CheckInterfaces", dev, "GigabitEthernet0/1: 33 input errors"}},
		},
		{
			"wrong IntfStatus: CRC errors",
			dev,
			map[string]string{"show interfaces": `GigabitEthernet0/1 is up, line protocol is up 
Description: bad because input errors
     0 input errors, 92 CRC, 0 frame, 0 overrun, 0 ignored, 0 abort
`},
			nil, // no error
			[]CheckResult{{"CheckInterfaces", dev, "GigabitEthernet0/1: 92 CRC errors"}},
		},
	}

	for _, test := range tests {
		got, gotErr := CheckInterfaces(test.Device, test.CmdResults)
		if gotErr != test.WantErr {
			t.Fatalf("CheckInterfaces error for test %q: %v WantErr %v", test.Comment, gotErr, test.WantErr)
		}

		if diff := deep.Equal(got, test.Want); diff != nil {
			t.Errorf("test %q: %v", test.Comment, diff)
		}
	}
}

func TestCheckBgpSum(t *testing.T) {
	dev := "router1"

	tests := []struct {
		Comment    string
		Device     string
		CmdResults map[string]string
		WantErr    error
		Want       []CheckResult
	}{
		{
			"healthy BGP session with numeric prefix count",
			dev,
			map[string]string{"show bgp sum": `BGP router identifier 192.0.2.70, local AS number 65550
BGP table version is 9, main routing table version 9

Neighbor        V    AS MsgRcvd MsgSent   TblVer  InQ OutQ Up/Down  State/PfxRcd
192.0.2.77      4 65551    6965    1766        9    0    0  5w4d           1
192.0.2.78      4 65552    6965    1766        9    0    0  5w4d          10
`},
			nil,
			nil, // No errors expected for healthy session
		},
		{
			"BGP session in Idle state",
			dev,
			map[string]string{"show bgp sum": `BGP router identifier 192.0.2.70, local AS number 65550
BGP table version is 9, main routing table version 9

Neighbor        V    AS MsgRcvd MsgSent   TblVer  InQ OutQ Up/Down  State/PfxRcd
192.0.2.77      4 65551       0       0        0    0    0 00:00:01 Idle
`},
			nil,
			[]CheckResult{{"CheckBgpSum", dev, "192.0.2.77: bad status \"Idle\""}},
		},
		{
			"BGP session in Active state",
			dev,
			map[string]string{"show bgp sum": `BGP router identifier 192.0.2.70, local AS number 65550

Neighbor        V    AS MsgRcvd MsgSent   TblVer  InQ OutQ Up/Down  State/PfxRcd
10.1.1.1        4 65003     100     100        0    0    0 00:01:23 Active
`},
			nil,
			[]CheckResult{{"CheckBgpSum", dev, "10.1.1.1: bad status \"Active\""}},
		},
		{
			"BGP session in Connect state",
			dev,
			map[string]string{"show bgp sum": `BGP router identifier 192.0.2.70, local AS number 65550

Neighbor        V    AS MsgRcvd MsgSent   TblVer  InQ OutQ Up/Down  State/PfxRcd
172.16.1.1      4 65004      50      50        0    0    0 00:00:45 Connect
`},
			nil,
			[]CheckResult{{"CheckBgpSum", dev, "172.16.1.1: bad status \"Connect\""}},
		},
		{
			"multiple BGP sessions with mixed states",
			dev,
			map[string]string{"show bgp sum": `BGP router identifier 192.0.2.70, local AS number 65550

Neighbor        V    AS MsgRcvd MsgSent   TblVer  InQ OutQ Up/Down  State/PfxRcd
192.0.2.77      4 65551    6965    1766        9    0    0  5w4d           1
192.0.2.78      4 65552       0       0        0    0    0 00:00:05 Idle
10.1.1.1        4 65553    5000    5000      123    0    0 2d12h          100
172.16.1.1      4 65554      10      10        0    0    0 00:00:30 Active
`},
			nil,
			[]CheckResult{
				{"CheckBgpSum", dev, "192.0.2.78: bad status \"Idle\""},
				{"CheckBgpSum", dev, "172.16.1.1: bad status \"Active\""},
			},
		},
		{
			"missing show bgp sum command output",
			dev,
			map[string]string{},
			nil,
			[]CheckResult{{"CheckBgpSum", dev, "failed to get 'show bgp sum' command output"}},
		},
	}

	for _, test := range tests {
		got, gotErr := CheckBgpSum(test.Device, test.CmdResults)
		if gotErr != test.WantErr {
			t.Fatalf("CheckBgpSum error for test %q: %v WantErr %v", test.Comment, gotErr, test.WantErr)
		}

		if diff := deep.Equal(got, test.Want); diff != nil {
			t.Errorf("test %q: %v", test.Comment, diff)
		}
	}
}
