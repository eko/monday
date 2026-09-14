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
	networkInterface    = ""
	networkInterfaceErr error

	// execCommand and lookPath are overridable in tests
	execCommand     = exec.Command
	defaultLookPath = exec.LookPath
	lookPath        = defaultLookPath

	// ErrAliasRemovalUnsupported is returned when an IP alias cannot be
	// removed precisely from the loopback interface on the current system
	ErrAliasRemovalUnsupported = errors.New("removing a loopback IP alias requires the 'ip' command on Linux")
)

// aliasAction is an operation on a loopback interface IP alias
type aliasAction int

const (
	aliasAdd aliasAction = iota
	aliasRemove
)

func init() {
	networkInterface, networkInterfaceErr = getNetworkInterface()
}

func getNetworkInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("cannot retrieve interfaces list: %v", err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return iface.Name, nil
		}
	}

	return "", errors.New("unable to find 'loopback' network interface")
}

// ListLoopbackAliases returns the IPv4 addresses currently assigned to the
// loopback interface in the range Monday allocates for its hostnames
// (127.0.1.0 and above), whatever the run that added them
func ListLoopbackAliases() ([]string, error) {
	if networkInterfaceErr != nil {
		return nil, networkInterfaceErr
	}

	iface, err := net.InterfaceByName(networkInterface)
	if err != nil {
		return nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	aliases := make([]string, 0)
	for _, addr := range addrs {
		if ip := addrIP(addr); isMondayLoopbackIP(ip) {
			aliases = append(aliases, ip.String())
		}
	}

	return aliases, nil
}

// RemoveLoopbackAlias removes an IP address previously added by Monday on the
// loopback interface
func RemoveLoopbackAlias(ip string) error {
	return runAliasCommand(aliasRemove, ip)
}

func addLoopbackAlias(ip string) error {
	return runAliasCommand(aliasAdd, ip)
}

func runAliasCommand(action aliasAction, ip string) error {
	if networkInterfaceErr != nil {
		return fmt.Errorf("unable to resolve the network interface to manage IP addresses: %w", networkInterfaceErr)
	}

	command, args, err := loopbackCommand(runtime.GOOS, action, networkInterface, ip)
	if err != nil {
		return err
	}

	output, err := execCommand(command, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"command '%s %s' failed on network interface '%s': %v: %s",
			command,
			strings.Join(args, " "),
			networkInterface,
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return nil
}

// loopbackCommand returns the command (and its arguments) adding or removing
// an IP alias on the loopback interface of the given operating system
func loopbackCommand(
	goos string,
	action aliasAction,
	iface string,
	ip string,
) (string, []string, error) {
	switch goos {
	case "darwin":
		if action == aliasAdd {
			return "ifconfig", []string{iface, "alias", ip, "up"}, nil
		}

		return "ifconfig", []string{iface, "-alias", ip}, nil

	case "linux":
		// iproute2 is preferred: it adds and removes a precise /32 address,
		// whereas net-tools' ifconfig replaces the interface primary address
		if _, err := lookPath("ip"); err == nil {
			verb := "add"
			if action == aliasRemove {
				verb = "del"
			}

			return "ip", []string{"addr", verb, ip + "/32", "dev", iface}, nil
		}

		if _, err := lookPath("ifconfig"); err == nil {
			if action == aliasAdd {
				return "ifconfig", []string{iface, ip, "up"}, nil
			}

			return "", nil, ErrAliasRemovalUnsupported
		}

		return "", nil, errors.New("neither 'ip' nor 'ifconfig' command is available to manage the loopback interface")

	default:
		return "", nil, fmt.Errorf("sorry, it seems your OS (%s) is not available yet", goos)
	}
}

// assignIpToPort finds, starting from the given address, the first loopback IP
// on which the given port is free, adding the IP alias to the interface when
// needed. It returns the address, the aliases it added along the way (so they
// can be removed later) and an error when no address is available.
func assignIpToPort(
	a, b, c, d byte,
	port string,
) (byte, byte, byte, byte, []string, error) {
	added := make([]string, 0)

	if networkInterfaceErr != nil {
		return a, b, c, d, added, fmt.Errorf("unable to resolve the network interface to attribute IP addresses: %w", networkInterfaceErr)
	}

	// Retrieve network interface
	iface, err := net.InterfaceByName(networkInterface)
	if err != nil {
		return a, b, c, d, added, err
	}

	for {
		// Maximum IP bytes reached
		if b == 255 && c == 255 && d == 255 {
			break
		}

		ip := net.IPv4(a, b, c, d)

		addrs, err := iface.Addrs()
		if err != nil {
			return a, b, c, d, added, err
		}

		// In case IP is already assigned to network interface, don't try to create it again
		if !isAlreadyAssigned(ip, addrs) {
			if err := addLoopbackAlias(ip.String()); err != nil {
				return a, b, c, d, added, err
			}

			added = append(added, ip.String())
		}

		// Can't be contacted on ip/port? it means this couple is free to be used
		if !canDial(ip.String(), port) {
			return a, b, c, d, added, nil
		}

		a, b, c, d = getNextIPAddress(a, b, c, d)
	}

	return a, b, c, d, added, fmt.Errorf("unable to find an available IP/Port (ip: %d.%d.%d.%d:%s)", a, b, c, d, port)
}

func isAlreadyAssigned(ip net.IP, addrs []net.Addr) bool {
	for _, addr := range addrs {
		if assigned := addrIP(addr); assigned != nil && assigned.Equal(ip) {
			return true
		}
	}

	return false
}

// isMondayLoopbackIP returns true for the loopback addresses Monday allocates:
// 127.0.1.0 and above, leaving the 127.0.0.x addresses untouched
func isMondayLoopbackIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil || ip4[0] != 127 {
		return false
	}

	return ip4[1] != 0 || ip4[2] != 0
}

func addrIP(addr net.Addr) net.IP {
	switch value := addr.(type) {
	case *net.IPNet:
		return value.IP
	case *net.IPAddr:
		return value.IP
	}

	return nil
}

func canDial(ip, port string) bool {
	conn, err := net.Dial("tcp", net.JoinHostPort(ip, port))
	if conn != nil {
		_ = conn.Close()
	}
	if err != nil {
		return false
	}

	return true
}
