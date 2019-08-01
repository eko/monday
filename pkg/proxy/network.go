package proxy

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

var (
	networkInterface = ""
)

func init() {
	var err error
	networkInterface, err = getNetworkInterface()
	if err != nil {
		panic(err)
	}
}

func getNetworkInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("Cannot retrieve interfaces list: ", err)
		return "", err
	}

	for _, i := range ifaces {
		ifaceFlags := i.Flags.String()
		if strings.Contains(ifaceFlags, "loopback") {
			return i.Name, nil
		}
	}

	return "", errors.New("Unable to find loopback network interface")
}

func generateIP(a byte, b byte, c byte, d int, port string) (net.IP, error) {
	ip := net.IPv4(a, b, c, byte(d))

	// Retrieve network interface
	iface, err := net.InterfaceByName(networkInterface)
	if err != nil {
		return net.IP{}, err
	}

	// Add a new IP address on the network interface
	command := "ifconfig"
	var args []string

	switch runtime.GOOS {
	case "darwin":
		args = []string{networkInterface, "alias", ip.String(), "up"}

	case "linux":
		args = []string{networkInterface, ip.String(), "up"}

	default:
		return net.IP{}, fmt.Errorf("Unable to find your OS: %s", runtime.GOOS)
	}

	for i := d; i < 255; i++ {
		ip = net.IPv4(a, b, c, byte(i))

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

		if err := exec.Command(command, args...).Run(); err != nil {
			return net.IP{}, fmt.Errorf("Cannot run ifconfig command to add new IP address (%s) on network interface '%s': %v", ip.String(), networkInterface, err)
		}

		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ip.String(), port))
		if err == nil {
			return net.IPv4(a, b, c, byte(i)), nil
		}
		if conn != nil {
			conn.Close()
		}
	}

	return net.IP{}, fmt.Errorf("Unable to find an available IP/Port (ip: %d.%d.%d.%d:%s)", a, b, c, d, port)
}
