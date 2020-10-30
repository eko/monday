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
		return "", fmt.Errorf("cannot retrieve interfaces list: %v", err)
	}

	for _, iface := range ifaces {
		ifaceFlags := iface.Flags.String()
		if strings.Contains(ifaceFlags, "loopback") {
			return iface.Name, nil
		}
	}

	return "", errors.New("unable to find 'loopback' network interface")
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

func assignIpToPort(a, b, c, d byte, port string) (byte, byte, byte, byte, error) {
	// Retrieve network interface
	iface, err := net.InterfaceByName(networkInterface)
	if err != nil {
		return a, b, c, d, err
	}

	for {
		// Maximum IP bytes reached
		if b == 255 && c == 255 && d == 255 {
			break
		}

		ip := net.IPv4(a, b, c, d)

		addrs, err := iface.Addrs()
		if err != nil {
			return a, b, c, d, err
		}

		// In case IP is already assigned to network interface, don't try to create it again
		if !isAlreadyAssigned(ip, addrs) {
			command, args := getAddIPCommandWithArgs(ip.String())

			if err := exec.Command(command, args...).Run(); err != nil {
				return a, b, c, d, fmt.Errorf("error while trying to run ifconfig/ip command to add new IP address (%s) on network interface '%s': %v", ip.String(), networkInterface, err)
			}
		}

		// Can't be contacted on ip/port? it means this couple is free to be used
		if !canDial(ip.String(), port) {
			return a, b, c, d, nil
		}

		a, b, c, d = getNextIPAddress(a, b, c, d)
	}

	return a, b, c, d, fmt.Errorf("unable to find an available IP/Port (ip: %d.%d.%d.%d:%s)", a, b, c, d, port)
}

func isAlreadyAssigned(ip net.IP, addrs []net.Addr) bool {
	for _, addr := range addrs {
		if addr.String() == ip.String()+"/8" {
			return true
		}
	}

	return false
}

func canDial(ip, port string) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
	if conn != nil {
		conn.Close()
	}
	if err != nil {
		return false
	}

	return true
}
