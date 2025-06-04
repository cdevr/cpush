package checks

import (
	"testing"

	"github.com/go-test/deep"
)

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
			"no command output",
			dev,
			map[string]string{}, // Missing "show bgp sum"
			nil,
			[]CheckResult{{"CheckBgpSum", dev, "failed to get 'show bgp sum' command output"}},
		},
		{
			"good bgp session - numeric state",
			dev,
			map[string]string{"show bgp sum": "Neighbor V AS MsgRcvd MsgSent TblVer InQ OutQ Up/Down State/PfxRcd\n1.1.1.1  4 2  1       1       0      0    0    00:00:03        1\n"},
			nil,
			nil, // Expect no issues
		},
		{
			"bad bgp session - Idle state",
			dev,
			map[string]string{"show bgp sum": "Neighbor V AS MsgRcvd MsgSent TblVer InQ OutQ Up/Down State/PfxRcd\n1.1.1.1  4 2  1       1       0      0    0    00:00:03        Idle\n"},
			nil,
			[]CheckResult{{"CheckBgpSum", dev, "1.1.1.1: idle status \"Idle\""}},
		},
		{
			"bad bgp session - Active state",
			dev,
			map[string]string{"show bgp sum": "Neighbor V AS MsgRcvd MsgSent TblVer InQ OutQ Up/Down State/PfxRcd\n2.2.2.2  4 2  0       0       0      0    0    00:00:00        Active\n"},
			nil,
			[]CheckResult{{"CheckBgpSum", dev, "2.2.2.2: idle status \"Active\""}},
		},
		{
			"mixed bgp sessions - one good, one bad",
			dev,
			map[string]string{"show bgp sum": "Neighbor V AS MsgRcvd MsgSent TblVer InQ OutQ Up/Down State/PfxRcd\n1.1.1.1  4 2  1       1       0      0    0    00:00:03        1\n2.2.2.2  4 2  0       0       0      0    0    00:00:00        Connect\n"},
			nil,
			[]CheckResult{{"CheckBgpSum", dev, "2.2.2.2: idle status \"Connect\""}},
		},
		{
			"empty bgp sum output",
			dev,
			map[string]string{"show bgp sum": "Neighbor V AS MsgRcvd MsgSent TblVer InQ OutQ Up/Down State/PfxRcd\n"}, // Just the header
			nil,
			nil, // Expect no issues as no neighbors to check
		},
	}

	for _, test := range tests {
		t.Run(test.Comment, func(t *testing.T) {
			got, gotErr := CheckBgpSum(test.Device, test.CmdResults)
			if gotErr != test.WantErr {
				t.Fatalf("CheckBgpSum error for test %q: %v WantErr %v", test.Comment, gotErr, test.WantErr)
			}

			if diff := deep.Equal(got, test.Want); diff != nil {
				t.Errorf("test %q: %v", test.Comment, diff)
			}
		})
	}
}
