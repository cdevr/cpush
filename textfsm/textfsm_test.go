package textfsm

import (
	"log"
	"os"
	"testing"

	"github.com/go-test/deep"
)

var (
	showInterfaceResult = []map[string]interface{}{
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

	typedShowInterfaceResult = []CiscoIosShowInterfacesRow{
		{
			Abort:          "",
			Bandwidth:      "100000 Kbit",
			BiaMacAddress:  "0001.961f.1b70",
			Crc:            "",
			Delay:          "100 usec",
			Description:    "",
			Duplex:         "Full-duplex",
			Encapsulation:  "ARPA",
			Frame:          "",
			Giants:         "",
			HardwareType:   "AmdFE",
			InputErrors:    "",
			InputPackets:   "",
			InputPps:       "",
			InputRate:      "",
			Intf:           "FastEthernet1/0",
			Ip:             "192.168.1.9",
			LastInput:      "",
			LastOutput:     "",
			LastOutputHang: "",
			LinkStatus:     "up",
			MacAddress:     "0001.961f.1b70",
			MediaType:      "",
			Mtu:            "1500",
			OutputErrors:   "",
			OutputPackets:  "",
			OutputPps:      "",
			OutputRate:     "",
			Overrun:        "",
			Prefixlen:      "28",
			ProtocolStatus: "down",
			QueueStrategy:  "",
			Runts:          "",
			Speed:          "100Mb/s",
			VlanId:         "",
			VlanIdInner:    "",
			VlanIdOuter:    "",
		},
	}

	showInterfaceResult2 = []map[string]interface{}{
		{
			"interface":        "ATM0",
			"description":      "descripting descriptions",
			"abort":            "0",
			"bandwidth":        "4608 Kbit",
			"bia_mac_address":  "80e0.1ded.6e8a",
			"crc":              "0",
			"delay":            "80 usec",
			"duplex":           "",
			"encapsulation":    "ATM",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "MPC",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "administratively down",
			"mac_address":      "80e0.1ded.6e8a",
			"media_type":       "",
			"mtu":              "1600",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "Per VC Queueing",
			"runts":            "0",
			"speed":            "",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"interface":        "BRI0",
			"description":      "",
			"abort":            "0",
			"bandwidth":        "64 Kbit",
			"bia_mac_address":  "",
			"crc":              "0",
			"delay":            "20000 usec",
			"duplex":           "",
			"encapsulation":    "HDLC",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "BRI with S",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "administratively down",
			"mac_address":      "",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"abort":            "0",
			"interface":        "BRI0:1",
			"description":      "",
			"bandwidth":        "64 Kbit",
			"bia_mac_address":  "",
			"crc":              "0",
			"delay":            "20000 usec",
			"duplex":           "",
			"encapsulation":    "HDLC",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "BRI",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "administratively down",
			"mac_address":      "",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"interface":        "BRI0:2",
			"description":      "",
			"abort":            "0",
			"bandwidth":        "64 Kbit",
			"bia_mac_address":  "",
			"crc":              "0",
			"delay":            "20000 usec",
			"duplex":           "",
			"encapsulation":    "HDLC",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "BRI",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "administratively down",
			"mac_address":      "",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"interface":        "Ethernet0",
			"description":      "",
			"abort":            "",
			"bandwidth":        "100000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e8e",
			"crc":              "0",
			"delay":            "100 usec",
			"duplex":           "",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "PQII_VDSL_ETHERNET",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "administratively down",
			"mac_address":      "80e0.1ded.6e8e",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{"abort": "",
			"interface":        "GigabitEthernet0",
			"description":      "Not Managed Interface <LI:C>",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e8b",
			"crc":              "0",
			"delay":            "10 usec",
			"duplex":           "Full-duplex",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Gigabit Ethernet",
			"input_errors":     "0",
			"input_packets":    "3624629564",
			"input_pps":        "575",
			"input_rate":       "959000",
			"ip":               "",
			"last_input":       "00:00:01",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "up",
			"mac_address":      "80e0.1ded.6e8b",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "608239333",
			"output_pps":       "596",
			"output_rate":      "2710000",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "up",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "1000Mb/s",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{"abort": "",
			"interface":        "GigabitEthernet1",
			"description":      "Not Managed Interface <LI:C>",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e8c",
			"crc":              "0",
			"delay":            "10 usec",
			"duplex":           "Auto-duplex",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Gigabit Ethernet",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "down",
			"mac_address":      "80e0.1ded.6e8c",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "Auto-speed",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{"abort": "",
			"interface":        "GigabitEthernet2",
			"description":      "Not Managed Interface <LI:C>",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e8d",
			"crc":              "0",
			"delay":            "10 usec",
			"duplex":           "Auto-duplex",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Gigabit Ethernet",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "down",
			"mac_address":      "80e0.1ded.6e8d",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "Auto-speed",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"interface":        "GigabitEthernet3",
			"description":      "Not Managed Interface <LI:C>",
			"abort":            "",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e8e",
			"crc":              "0",
			"delay":            "10 usec",
			"duplex":           "Auto-duplex",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Gigabit Ethernet",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "down",
			"mac_address":      "80e0.1ded.6e8e",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "Auto-speed",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{"abort": "",
			"interface":        "GigabitEthernet4",
			"description":      "Not Managed Interface <LI:C>",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e8f",
			"crc":              "0",
			"delay":            "10 usec",
			"duplex":           "Auto-duplex",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Gigabit Ethernet",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "down",
			"mac_address":      "80e0.1ded.6e8f",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "Auto-speed",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"interface":        "GigabitEthernet5",
			"description":      "Not Managed Interface <LI:C>",
			"abort":            "",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e90",
			"crc":              "0",
			"delay":            "10 usec",
			"duplex":           "Auto-duplex",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Gigabit Ethernet",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "down",
			"mac_address":      "80e0.1ded.6e90",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "Auto-speed",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{"abort": "",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e91",
			"crc":              "0",
			"delay":            "10 usec",
			"description":      "Not Managed Interface <LI:C>",
			"duplex":           "Auto-duplex",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Gigabit Ethernet",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"interface":        "GigabitEthernet6",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "down",
			"mac_address":      "80e0.1ded.6e91",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "Auto-speed",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{"abort": "",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e92",
			"crc":              "0",
			"delay":            "10 usec",
			"description":      "Not Managed Interface <LI:C>",
			"duplex":           "Auto-duplex",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Gigabit Ethernet",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"interface":        "GigabitEthernet7",
			"ip":               "",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "down",
			"mac_address":      "80e0.1ded.6e92",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "down",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "Auto-speed",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{"abort": "",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e9c",
			"crc":              "0",
			"delay":            "10 usec",
			"description":      "WAN:ipd-olt730-r-ac-01:ge-1/0/6 <LI:L>",
			"duplex":           "Full Duplex",
			"encapsulation":    "802.1Q Virtual LAN",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "PQ3_TSEC",
			"input_errors":     "0",
			"input_packets":    "4934548363",
			"input_pps":        "601",
			"input_rate":       "2703000",
			"interface":        "GigabitEthernet8",
			"ip":               "",
			"last_input":       "00:00:00",
			"last_output":      "00:00:00",
			"last_output_hang": "never",
			"link_status":      "up",
			"mac_address":      "80e0.1ded.6e9c",
			"media_type":       "LX",
			"mtu":              "1500",
			"output_errors":    "0",
			"output_packets":   "3632965400",
			"output_pps":       "579",
			"output_rate":      "961000",
			"overrun":          "0",
			"prefixlen":        "",
			"protocol_status":  "up",
			"queue_strategy":   "Class-based queueing",
			"runts":            "0",
			"speed":            "1Gbps",
			"vlan_id":          "1",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"interface":        "GigabitEthernet8.300",
			"description":      "some description goes here",
			"abort":            "",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e9c",
			"crc":              "",
			"delay":            "10 usec",
			"duplex":           "",
			"encapsulation":    "802.1Q Virtual LAN",
			"frame":            "",
			"giants":           "",
			"hardware_type":    "PQ3_TSEC",
			"input_errors":     "",
			"input_packets":    "",
			"input_pps":        "",
			"input_rate":       "",
			"ip":               "172.17.85.109",
			"last_input":       "",
			"last_output":      "",
			"last_output_hang": "",
			"link_status":      "up",
			"mac_address":      "80e0.1ded.6e9c",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "",
			"output_packets":   "",
			"output_pps":       "",
			"output_rate":      "",
			"overrun":          "",
			"prefixlen":        "30",
			"protocol_status":  "up",
			"queue_strategy":   "",
			"runts":            "",
			"speed":            "",
			"vlan_id":          "300",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"interface":        "Loopback0",
			"description":      "",
			"abort":            "0",
			"bandwidth":        "8000000 Kbit",
			"bia_mac_address":  "",
			"crc":              "0",
			"delay":            "5000 usec",
			"duplex":           "",
			"encapsulation":    "LOOPBACK",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "Loopback",
			"input_errors":     "0",
			"input_packets":    "0",
			"input_pps":        "0",
			"input_rate":       "0",
			"ip":               "172.17.103.201",
			"last_input":       "never",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "up",
			"mac_address":      "",
			"media_type":       "",
			"mtu":              "1514",
			"output_errors":    "0",
			"output_packets":   "0",
			"output_pps":       "0",
			"output_rate":      "0",
			"overrun":          "0",
			"prefixlen":        "32",
			"protocol_status":  "up",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    ""},
		{
			"interface":        "Vlan1",
			"description":      "another description goes here",
			"abort":            "",
			"bandwidth":        "1000000 Kbit",
			"bia_mac_address":  "80e0.1ded.6e8a",
			"crc":              "0",
			"delay":            "10 usec",
			"duplex":           "",
			"encapsulation":    "ARPA",
			"frame":            "0",
			"giants":           "0",
			"hardware_type":    "EtherSVI",
			"input_errors":     "0",
			"input_packets":    "3616087870",
			"input_pps":        "577",
			"input_rate":       "941000",
			"ip":               "192.168.156.85",
			"last_input":       "00:00:01",
			"last_output":      "never",
			"last_output_hang": "never",
			"link_status":      "up",
			"mac_address":      "80e0.1ded.6e8a",
			"media_type":       "",
			"mtu":              "1500",
			"output_errors":    "",
			"output_packets":   "4902093344",
			"output_pps":       "596",
			"output_rate":      "2677000",
			"overrun":          "0",
			"prefixlen":        "29",
			"protocol_status":  "up",
			"queue_strategy":   "fifo",
			"runts":            "0",
			"speed":            "",
			"vlan_id":          "",
			"vlan_id_inner":    "",
			"vlan_id_outer":    "",
		},
	}
)

func TestShortCut(t *testing.T) {
	dataFn := "testdata/cisco_ios_show_interfaces"
	data := ReadFile(dataFn, t)

	got, err := ParseCiscoIosShowInterfaces(data)
	if err != nil {
		t.Errorf("textfsm failed to execute: %v", err)
	}

	want := showInterfaceResult

	if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}

func TestTypedShortCut(t *testing.T) {
	dataFn := "testdata/cisco_ios_show_interfaces"
	data := ReadFile(dataFn, t)

	got, err := ParseTypedCiscoIosShowInterfaces(data)
	if err != nil {
		t.Errorf("textfsm failed to execute: %v", err)
	}

	want := typedShowInterfaceResult

	if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
		t.Logf("%#v", got)
	}
}

func TestTextFSM(t *testing.T) {
	templateFn := "templates/cisco_ios_show_interfaces.textfsm"
	template := ReadFile(templateFn, t)

	dataFn := "testdata/cisco_ios_show_interfaces"
	data := ReadFile(dataFn, t)

	got, err := Parse(template, data, true)
	if err != nil {
		t.Errorf("textfsm failed to execute: %v", err)
	}

	want := showInterfaceResult

	if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}

func TestTextFSMLongerExample(t *testing.T) {
	templateFn := "templates/cisco_ios_show_interfaces.textfsm"
	template := ReadFile(templateFn, t)

	dataFn := "testdata/cisco_ios_show_interfaces2"
	data := ReadFile(dataFn, t)

	got, err := Parse(template, data, true)
	if err != nil {
		t.Errorf("textfsm failed to execute: %v", err)
	}

	want := showInterfaceResult2

	if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
		t.Logf("%#v", got)
	}
}

func ReadFile(fn string, t *testing.T) string {
	templateBytes, err := os.ReadFile(fn)
	if err != nil {
		t.Errorf("failed to read file at %q: %v", fn, err)
	}
	return string(templateBytes)
}

func TestExample(t *testing.T) {
	templateFn := "testdata/example.textfsm"
	template := ReadFile(templateFn, t)
	fsm, err := NewTextFSM(template)
	if err != nil {
		t.Fatalf("failed to parse example template at %q: %v", templateFn, err)
	}

	exampleFn := "testdata/example"
	example := ReadFile(exampleFn, t)
	got, err := fsm.Parse(example, true)
	if err != nil {
		t.Fatalf("failed to parse example input: %v", err)
	}

	want := []map[string]interface{}{
		{
			"Heading": "heading",
			"Detail":  []string{"detail1", "detail2"},
		},
		{
			"Heading": "heading2",
			"Detail":  []string{"detail3", "detail4"},
		},
	}

	if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
	}
}

type exampleRow struct {
	Heading string
	Detail  []string
}

func TestExampleStruct(t *testing.T) {
	templateFn := "testdata/example.textfsm"
	template := ReadFile(templateFn, t)

	fsm, err := NewTextFSM(template)
	if err != nil {
		t.Fatalf("failed to parse example template at %q: %v", templateFn, err)
	}

	exampleFn := "testdata/example"
	example := ReadFile(exampleFn, t)

	var data []exampleRow
	val, err := fsm.ParseToStruct(data, example, true)
	data = val.([]exampleRow)

	if err != nil {
		t.Fatalf("failed to parse example input: %v", err)
	}

	want := []exampleRow{
		{
			Heading: "heading",
			Detail:  []string{"detail1", "detail2"},
		},
		{
			Heading: "heading2",
			Detail:  []string{"detail3", "detail4"},
		},
	}

	if diff := deep.Equal(data, want); diff != nil {
		t.Error(diff)
		t.Logf("%#v", data)
	}
}

var showBgpSummaryResult = []CiscoIosShowBgpSummaryRow{
	{LocalAs: "65550", ReceivedV4: "1", RemoteAs: "65551", RemoteIp: "192.0.2.77", RouterId: "192.0.2.70", Status: "", Uptime: "5w4d"},
	{LocalAs: "65550", ReceivedV4: "10", RemoteAs: "65552", RemoteIp: "192.0.2.78", RouterId: "192.0.2.70", Status: "", Uptime: "5w4d"},
}

func TestShowBgpSum(t *testing.T) {
	dataFn := "testdata/cisco_ios_show_bgp_summary"
	data := ReadFile(dataFn, t)

	got, err := ParseTypedCiscoIosShowBgpSummary(data)
	if err != nil {
		t.Errorf("textfsm failed to execute: %v", err)
	}

	want := showBgpSummaryResult

	if diff := deep.Equal(got, want); diff != nil {
		t.Error(diff)
		log.Printf("got: %#v", got)
	}
}
