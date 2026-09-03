// jsontext reads the 2 text interfaces of encoding through encoding/json.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type celsius float64

func (c celsius) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%.1fC", float64(c))), nil
}

func (c *celsius) UnmarshalText(text []byte) error {
	var v float64
	if _, err := fmt.Sscanf(string(text), "%fC", &v); err != nil {
		return err
	}
	*c = celsius(v)
	return nil
}

type host struct {
	Addr net.IP           `json:"addr"`
	Net  *net.IPNet       `json:"net"`
	Temp celsius          `json:"temp"`
	Seen time.Time        `json:"seen"`
	Tags map[celsius]int  `json:"tags"`
	MAC  net.HardwareAddr `json:"mac"`
	List []net.IP         `json:"list"`
}

func main() {
	_, cidr, _ := net.ParseCIDR("10.0.0.0/24")
	mac, _ := net.ParseMAC("01:23:45:67:89:ab")
	h := host{
		Addr: net.ParseIP("192.168.0.1"),
		Net:  cidr,
		Temp: celsius(21.5),
		Seen: time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC),
		Tags: map[celsius]int{celsius(1): 7},
		MAC:  mac,
		List: []net.IP{net.ParseIP("::1")},
	}
	out, err := json.Marshal(h)
	fmt.Println(string(out), err)

	var back host
	err = json.Unmarshal(out, &back)
	fmt.Println(err)
	fmt.Println(back.Addr, back.Net, back.Temp, back.Seen.Format(time.RFC3339))
	fmt.Println(back.Tags[celsius(1)], back.MAC, back.List[0])

	bad := json.Unmarshal([]byte(`{"temp":5}`), &back)
	fmt.Println("a number for a text value:", bad != nil)
}
