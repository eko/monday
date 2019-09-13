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

// getAddIPCommandWithArgs returns the command (ifconfig, ip, ...) that will be used for the current OS
// and its associated arguments
func getAddIPCommandWithArgs(ip string) (string, []string) {
	var command = "ifconfig"
	var args []string

	switch runtime.GOOS {
	case "darwin":
		args = []string{networkInterface, "alias", ip, "up"}

	case "linux":
		// Check if "ifconfig" is available
		_, err := exec.LookPath(command)
		if err == nil {
			// "ifconfig" case
			args = []string{networkInterface, ip, "up"}
		} else {
			// "ip" case
			command = "ip"
			args = []string{"addr", "add", ip + "/32", "dev", networkInterface}
		}

	default:
		panic(fmt.Sprintf("Sorry, it seems your OS (%s) is not available yet.", runtime.GOOS))
	}

	return command, args
}

func generateIP(a byte, b byte, c byte, d int, port string) (net.IP, error) {
	ip := net.IPv4(a, b, c, byte(d))

	// Retrieve network interface
	iface, err := net.InterfaceByName(networkInterface)
	if err != nil {
		return net.IP{}, err
	}

	command, args := getAddIPCommandWithArgs(ip.String())

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
				if conn != nil {
					conn.Close()
				}
			}
		}

		// No already assigned IP/Port available, add IP address to the network interface
		if err := exec.Command(command, args...).Run(); err != nil {
			return net.IP{}, fmt.Errorf("Cannot run ifconfig command to add new IP address (%s) on network interface '%s': %v", ip.String(), networkInterface, err)
		}

		// Try to assign port to the newly assigned IP
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ip.String(), port))
		if err != nil {
			return net.IPv4(a, b, c, byte(i)), nil
		}
		if conn != nil {
			conn.Close()
		}
	}

	return net.IP{}, fmt.Errorf("Unable to find an available IP/Port (ip: %d.%d.%d.%d:%s)", a, b, c, d, port)
}
