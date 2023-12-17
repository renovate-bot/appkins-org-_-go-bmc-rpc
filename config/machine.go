package config

type Machine struct {
	MacAddress string `json:"mac_address"`
	PortIdx    string `json:"port_idx"`
	Host       string `json:"host"`
}
