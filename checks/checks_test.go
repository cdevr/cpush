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
			[]CheckResult{{"CheckInterfaces.IntfStatus", dev, "GigabitEthernet0/1: admin \"up\" protocol \"down\""}},
		},
		{
			"wrong IntfStatus: Input errors",
			dev,
			map[string]string{"show interfaces": `GigabitEthernet0/1 is up, line protocol is up 
Description: bad because input errors
      33 input errors, 0 CRC, 0 frame, 0 overrun, 0 ignored, 0 abort
`},
			nil, // no error
			[]CheckResult{{"CheckInterfaces.IntfStatus", dev, "GigabitEthernet0/1: 33 input errors"}},
		},
		{
			"wrong IntfStatus: CRC errors",
			dev,
			map[string]string{"show interfaces": `GigabitEthernet0/1 is up, line protocol is up 
Description: bad because input errors
     0 input errors, 92 CRC, 0 frame, 0 overrun, 0 ignored, 0 abort
`},
			nil, // no error
			[]CheckResult{{"CheckInterfaces.IntfStatus", dev, "GigabitEthernet0/1: 92 CRC errors"}},
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
