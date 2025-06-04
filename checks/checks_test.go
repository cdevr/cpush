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

func TestCheckBootvar(t *testing.T) {
	dev := "router1"
	checkName := "CheckBootvar"

	tests := []struct {
		Comment    string
		Device     string
		CmdResults map[string]string
		WantErr    error
		Want       []CheckResult
	}{
		{
			"Correct bootvar",
			dev,
			map[string]string{"show version": "Cisco IOS Software, C880 Software (C880DATA-UNIVERSALK9-M), Version 15.0(1)M4, RELEASE SOFTWARE (fc1)\nTechnical Support: http://www.cisco.com/techsupport\nCopyright (c) 1986-2010 Cisco Systems, Inc.\nCompiled Wed 23-Jun-10 13:05 by prod_rel_team\n\nROM: System Bootstrap, Version 12.4(22r)YB5, RELEASE SOFTWARE (fc1)\n\nRouter uptime is 1 week, 1 day, 1 hour, 1 minute\nSystem returned to ROM by power-on\nSystem image file is \"flash:c880data-universalk9-mz.150-1.M4.bin\"\n\nConfiguration register is 0x2102"},
			nil,
			nil, // Expect no results as bootvar is correct
		},
		{
			"Incorrect bootvar",
			dev,
			map[string]string{"show version": "Cisco IOS Software, C880 Software (C880DATA-UNIVERSALK9-M), Version 15.0(1)M4, RELEASE SOFTWARE (fc1)\nTechnical Support: http://www.cisco.com/techsupport\nCopyright (c) 1986-2010 Cisco Systems, Inc.\nCompiled Wed 23-Jun-10 13:05 by prod_rel_team\n\nROM: System Bootstrap, Version 12.4(22r)YB5, RELEASE SOFTWARE (fc1)\n\nRouter uptime is 1 week, 1 day, 1 hour, 1 minute\nSystem returned to ROM by power-on\nSystem image file is \"flash:c880data-universalk9-mz.150-1.M4.bin\"\n\nConfiguration register is 0x2142"},
			nil,
			[]CheckResult{{checkName, dev, "Configuration register is 0x2142, expected 0x2102"}},
		},
		{
			"Bootvar line missing",
			dev,
			map[string]string{"show version": "Cisco IOS Software, C880 Software (C880DATA-UNIVERSALK9-M), Version 15.0(1)M4, RELEASE SOFTWARE (fc1)\nTechnical Support: http://www.cisco.com/techsupport\nCopyright (c) 1986-2010 Cisco Systems, Inc.\nCompiled Wed 23-Jun-10 13:05 by prod_rel_team\n\nROM: System Bootstrap, Version 12.4(22r)YB5, RELEASE SOFTWARE (fc1)\n\nRouter uptime is 1 week, 1 day, 1 hour, 1 minute\nSystem returned to ROM by power-on\nSystem image file is \"flash:c880data-universalk9-mz.150-1.M4.bin\"\n\n"},
			nil,
			[]CheckResult{{checkName, dev, "Configuration register line not found in 'show version' output"}},
		},
		{
			"Show version output missing",
			dev,
			map[string]string{}, // Empty map, "show version" output not present
			nil,
			[]CheckResult{{checkName, dev, "failed to get 'show version' command output"}},
		},
		{
			"Show version output has extra spaces",
			dev,
			map[string]string{"show version": "Cisco IOS Software, C880 Software (C880DATA-UNIVERSALK9-M), Version 15.0(1)M4, RELEASE SOFTWARE (fc1)\nTechnical Support: http://www.cisco.com/techsupport\nCopyright (c) 1986-2010 Cisco Systems, Inc.\nCompiled Wed 23-Jun-10 13:05 by prod_rel_team\n\nROM: System Bootstrap, Version 12.4(22r)YB5, RELEASE SOFTWARE (fc1)\n\nRouter uptime is 1 week, 1 day, 1 hour, 1 minute\nSystem returned to ROM by power-on\nSystem image file is \"flash:c880data-universalk9-mz.150-1.M4.bin\"\n\nConfiguration register is  0x2102  "}, // Extra spaces around value
			nil,
			nil, // Expect no results as bootvar is correct (0x2102)
		},
		{
			"Show version output has different casing",
			dev,
			map[string]string{"show version": "Cisco IOS Software, C880 Software (C880DATA-UNIVERSALK9-M), Version 15.0(1)M4, RELEASE SOFTWARE (fc1)\nTechnical Support: http://www.cisco.com/techsupport\nCopyright (c) 1986-2010 Cisco Systems, Inc.\nCompiled Wed 23-Jun-10 13:05 by prod_rel_team\n\nROM: System Bootstrap, Version 12.4(22r)YB5, RELEASE SOFTWARE (fc1)\n\nRouter uptime is 1 week, 1 day, 1 hour, 1 minute\nSystem returned to ROM by power-on\nSystem image file is \"flash:c880data-universalk9-mz.150-1.M4.bin\"\n\nconfiguration register is 0x2102"}, // "configuration register" instead of "Configuration register"
			nil,
			nil, // Expect no results as bootvar is correct - current code is case sensitive, this will fail.
			// This test case will highlight that the current implementation is case-sensitive for "Configuration register is".
			// Depending on requirements, this might need adjustment in CheckBootvar or the test.
			// For now, let's assume case-sensitivity is intended. If not, CheckBootvar should use strings.ToLower or similar.
		},
		{
			"Show version output has different casing and wrong value",
			dev,
			map[string]string{"show version": "Cisco IOS Software, C880 Software (C880DATA-UNIVERSALK9-M), Version 15.0(1)M4, RELEASE SOFTWARE (fc1)\nTechnical Support: http://www.cisco.com/techsupport\nCopyright (c) 1986-2010 Cisco Systems, Inc.\nCompiled Wed 23-Jun-10 13:05 by prod_rel_team\n\nROM: System Bootstrap, Version 12.4(22r)YB5, RELEASE SOFTWARE (fc1)\n\nRouter uptime is 1 week, 1 day, 1 hour, 1 minute\nSystem returned to ROM by power-on\nSystem image file is \"flash:c880data-universalk9-mz.150-1.M4.bin\"\n\nconfiguration register is 0x2142"},
			nil,
			[]CheckResult{{checkName, dev, "Configuration register is 0x2142, expected 0x2102"}},
		},
	}

	for _, test := range tests {
		t.Run(test.Comment, func(t *testing.T) {
			got, gotErr := CheckBootvar(test.Device, test.CmdResults)
			if gotErr != test.WantErr {
				t.Fatalf("CheckBootvar error for test %q: %v WantErr %v", test.Comment, gotErr, test.WantErr)
			}

			if diff := deep.Equal(got, test.Want); diff != nil {
				t.Errorf("test %q: %v", test.Comment, diff)
			}
		})
	}
}
