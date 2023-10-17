package textfsm

const CiscoIosShowInterfacesTemplate = "Value Required interface (\\S+)\nValue link_status (.+?)\nValue protocol_status (.+?)\nValue hardware_type ([\\w ]+)\nValue mac_address ([a-fA-F0-9]{4}\\.[a-fA-F0-9]{4}\\.[a-fA-F0-9]{4})\nValue bia_mac_address ([a-fA-F0-9]{4}\\.[a-fA-F0-9]{4}\\.[a-fA-F0-9]{4})\nValue description (.+?)\nValue ip (\\d+\\.\\d+\\.\\d+\\.\\d+)\nValue prefixlen (\\d+)\nValue mtu (\\d+)\nValue duplex (([Ff]ull|[Aa]uto|[Hh]alf|[Aa]-).*?)\nValue speed (.*?)\nValue media_type (\\S+.*)\nValue bandwidth (\\d+\\s+\\w+)\nValue delay (\\d+\\s+\\S+)\nValue encapsulation (.+?)\nValue last_input (.+?)\nValue last_output (.+?)\nValue last_output_hang (.+?)\nValue queue_strategy (.+)\nValue input_rate (\\d+)\nValue output_rate (\\d+)\nValue input_pps (\\d+)\nValue output_pps (\\d+)\nValue input_packets (\\d+)\nValue output_packets (\\d+)\nValue runts (\\d+)\nValue giants (\\d+)\nValue input_errors (\\d+)\nValue crc (\\d+)\nValue frame (\\d+)\nValue overrun (\\d+)\nValue abort (\\d+)\nValue output_errors (\\d+)\nValue vlan_id (\\d+)\nValue vlan_id_inner (\\d+)\nValue vlan_id_outer (\\d+)\n\nStart\n  ^\\S+\\s+is\\s+.+?,\\s+line\\s+protocol.*$$ -> Continue.Record\n  ^${interface}\\s+is\\s+${link_status},\\s+line\\s+protocol\\s+is\\s+${protocol_status}\\s*$$\n  ^\\s+Hardware\\s+is\\s+${hardware_type} -> Continue\n  ^.+address\\s+is\\s+${mac_address}\\s+\\(bia\\s+${bia_mac_address}\\)\\s*$$\n  ^\\s+Description:\\s+${description}\\s*$$\n  ^\\s+Internet\\s+address\\s+is\\s+${ip}\\/${prefixlen}\\s*$$\n  ^\\s+MTU\\s+${mtu}.*BW\\s+${bandwidth}.*DLY\\s+${delay},\\s*$$\n  ^\\s+Encapsulation\\s+${encapsulation}, Vlan ID\\s+${vlan_id}.+$$\n  ^\\s+Encapsulation\\s+${encapsulation}, outer ID\\s+${vlan_id_outer}, inner ID\\s+${vlan_id_inner}.+$$\n  ^\\s+Encapsulation\\s+${encapsulation},.+$$\n  ^\\s+Last\\s+input\\s+${last_input},\\s+output\\s+${last_output},\\s+output\\s+hang\\s+${last_output_hang}\\s*$$\n  ^\\s+Queueing\\s+strategy:\\s+${queue_strategy}\\s*$$\n  ^\\s+${duplex},\\s+${speed},.+media\\stype\\sis\\s${media_type}$$\n  ^\\s+${duplex},\\s+${speed},.+TX/FX$$\n  ^\\s+${duplex},\\s+${speed}$$\n  ^.*input\\s+rate\\s+${input_rate}\\s+\\w+/sec,\\s+${input_pps}\\s+packets.+$$\n  ^.*output\\s+rate\\s+${output_rate}\\s+\\w+/sec,\\s+${output_pps}\\s+packets.+$$\n  ^\\s+${input_packets}\\s+packets\\s+input,\\s+\\d+\\s+bytes,\\s+\\d+\\s+no\\s+buffer\\s*$$\n  ^\\s+${runts}\\s+runts,\\s+${giants}\\s+giants,\\s+\\d+\\s+throttles\\s*$$\n  ^\\s+${input_errors}\\s+input\\s+errors,\\s+${crc}\\s+(crc|CRC),\\s+${frame}\\s+frame,\\s+${overrun}\\s+overrun,\\s+\\d+\\s+ignored\\s*$$\n  ^\\s+${input_errors}\\s+input\\s+errors,\\s+${crc}\\s+(crc|CRC),\\s+${frame}\\s+frame,\\s+${overrun}\\s+overrun,\\s+\\d+\\s+ignored,\\s+${abort}\\s+abort\\s*$$\n  ^\\s+${output_packets}\\s+packets\\s+output,\\s+\\d+\\s+bytes,\\s+\\d+\\s+underruns\\s*$$\n  ^\\s+${output_errors}\\s+output\\s+errors,\\s+\\d+\\s+collisions,\\s+\\d+\\s+interface\\s+resets\\s*$$\n  # Capture time-stamp if vty line has command time-stamping turned on\n  ^Load\\s+for\\s+\n  ^Time\\s+source\\s+is\n"

func ParseCiscoIosShowInterfaces(input string) ([]map[string]interface{}, error) {
	return Parse(CiscoIosShowInterfacesTemplate, input, true)
}

type CiscoIosShowInterfacesRow struct {
	Abort          string
	Bandwidth      string
	BiaMacAddress  string
	Crc            string
	Delay          string
	Description    string
	Duplex         string
	Encapsulation  string
	Frame          string
	Giants         string
	HardwareType   string
	InputErrors    string
	InputPackets   string
	InputPps       string
	InputRate      string
	Intf           string
	Ip             string
	LastInput      string
	LastOutput     string
	LastOutputHang string
	LinkStatus     string
	MacAddress     string
	MediaType      string
	Mtu            string
	OutputErrors   string
	OutputPackets  string
	OutputPps      string
	OutputRate     string
	Overrun        string
	Prefixlen      string
	ProtocolStatus string
	QueueStrategy  string
	Runts          string
	Speed          string
	VlanId         string
	VlanIdInner    string
	VlanIdOuter    string
}

func ParseTypedCiscoIosShowInterfaces(input string) ([]CiscoIosShowInterfacesRow, error) {
	result, err := ParseIntoStruct([]CiscoIosShowInterfacesRow{}, CiscoIosShowInterfacesTemplate, input, true)
	return result.([]CiscoIosShowInterfacesRow), err
}

const ExampleTemplate = "Value Heading ([^\\s].*)\nValue List Detail (.*)\n\nStart\n  ^${Heading} -> heading\n\nheading\n  ^\\s${Detail}\n  # If you find a new heading, don't yet read it into the \"heading\" field, first record it.\n  ^.* -> Continue.Record\n  ^${Heading}\n"

func ParseExample(input string) ([]map[string]interface{}, error) {
	return Parse(ExampleTemplate, input, true)
}

type ExampleRow struct {
	Detail  []string
	Heading string
}

func ParseTypedExample(input string) ([]ExampleRow, error) {
	result, err := ParseIntoStruct([]ExampleRow{}, ExampleTemplate, input, true)
	return result.([]ExampleRow), err
}
