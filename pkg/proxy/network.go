package proxy

import (
	"fmt"
	"net"
	"os/exec"
)

const (
	networkInterface = "lo0"
)

func generateIP(a byte, b byte, c byte, d int, port string) (net.IP, error) {
	ip := net.IPv4(a, b, c, byte(d))

	for i := d; i < 255; i++ {
		ip = net.IPv4(a, b, c, byte(i))

		// Check lo0 interface exists
		iface, err := net.InterfaceByName(networkInterface)
		if err != nil {
			return net.IP{}, err
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return net.IP{}, err
		}

		// Try to assign port to the IP addresses already assigned to the interface
		for _, addr := range addrs {
			if addr.String() == ip.String()+"/8" {
				conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ip.String(), port))
				if err != nil {
					return net.IPv4(a, b, c, byte(i)), nil
				}
				conn.Close()
			}
		}

		// Add a new IP address on the network interface
		command := "ifconfig"
		args := []string{"lo0", "alias", ip.String(), "up"}
		if err := exec.Command(command, args...).Run(); err != nil {
			return net.IP{}, fmt.Errorf("Cannot run ifconfig command to add new IP address (%s) on lo0 interface: %v", ip.String(), err)
		}

		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ip.String(), port))
		if err == nil {
			return net.IPv4(a, b, c, byte(i)), nil
		}
		conn.Close()
	}

	return net.IP{}, fmt.Errorf("Unable to find an available IP/Port (ip: %d.%d.%d.%d:%s)", a, b, c, d, port)
}
