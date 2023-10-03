package textfsm

import (
	"os"
	"testing"

	"github.com/go-test/deep"
)

func TestTextFSM(t *testing.T) {
	templateFn := "templates/cisco_ios_show_interfaces.textfsm"
	templateBytes, err := os.ReadFile(templateFn)
	if err != nil {
		t.Errorf("failed to read template at %q: %v", templateFn, err)
	}
	template := string(templateBytes)

	dataFn := "testdata/cisco_ios_show_interfaces"
	dataBytes, err := os.ReadFile(dataFn)
	if err != nil {
		t.Errorf("failed to read template data at %q: %v", dataFn, err)
	}
	data := string(dataBytes)

	fsm := TextFSM{}
	err = fsm.ParseString(template)
	if err != nil {
		t.Errorf("textfsm failed to parse template: %v", err)
	}

	parser := ParserOutput{}
	err = parser.ParseTextString(data, fsm, true)
	if err != nil {
		t.Errorf("failed to parse input data: %v", err)
	}

	want := []map[string]interface{}{
		{
			"interface":        "FastEthernet1/0",
			"description":      "",
			"link_status":      "up",
			"protocol_status":  "down",
			"ip":               "192.168.1.9",
			"prefixlen":        "28",
			"mac_address":      "0001.961f.1b70",
			"bia_mac_address":  "0001.961f.1b70",
			"mtu":              "1500",
			"delay":            "100 usec",
			"bandwidth":        "100000 Kbit",
			"speed":            "100Mb/s",
			"duplex":           "Full-duplex",
			"encapsulation":    "ARPA",
			"hardware_type":    "AmdFE",
			"media_type":       "",
			"overrun":          "",
			"last_output":      "",
			"last_output_hang": "",
			"input_packets":    "",
			"input_rate":       "",
			"input_pps":        "",
			"input_errors":     "",
			"output_packets":   "",
			"output_rate":      "",
			"output_pps":       "",
			"output_errors":    "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    "",
			"crc":              "",
			"abort":            "",
			"giants":           "",
			"last_input":       "",
			"queue_strategy":   "",
			"frame":            "",
			"vlan_id":          "",
			"runts":            "",
		},
	}
	got := parser.Dict

	if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}
