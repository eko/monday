package txeh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

const UNKNOWN = 0
const EMPTY = 10
const COMMENT = 20
const ADDRESS = 30

type IPFamily int64

const (
	IPFamilyV4 IPFamily = iota
	IPFamilyV6
)

type HostsConfig struct {
	ReadFilePath  string
	WriteFilePath string
	// RawText for input. If RawText is set ReadFilePath, WriteFilePath are ignored. Use RenderHostsFile rather
	// than save to get the results.
	RawText *string
}

type Hosts struct {
	sync.Mutex
	*HostsConfig
	hostFileLines HostFileLines
}

// AddressLocations the location of an address in the HFL
type AddressLocations map[string]int

// HostLocations maps a hostname
// to an original line number
type HostLocations map[string]int

type HostFileLines []HostFileLine

type HostFileLine struct {
	OriginalLineNum int
	LineType        int
	Address         string
	Parts           []string
	Hostnames       []string
	Raw             string
	Trimmed         string
	Comment         string
}

// NewHostsDefault returns a hosts object with
// default configuration
func NewHostsDefault() (*Hosts, error) {
	return NewHosts(&HostsConfig{})
}

// NewHosts returns a new hosts object
func NewHosts(hc *HostsConfig) (*Hosts, error) {
	h := &Hosts{HostsConfig: hc}
	h.Lock()
	defer h.Unlock()

	defaultHostsFile := "/etc/hosts"

	if runtime.GOOS == "windows" {
		defaultHostsFile = `C:\Windows\System32\Drivers\etc\hosts`
	}

	if h.ReadFilePath == "" && h.RawText == nil {
		h.ReadFilePath = defaultHostsFile
	}

	if h.WriteFilePath == "" && h.RawText == nil {
		h.WriteFilePath = h.ReadFilePath
	}

	if h.RawText != nil {
		hfl, err := ParseHostsFromString(*h.RawText)
		if err != nil {
			return nil, err
		}

		h.hostFileLines = hfl
		return h, nil
	}

	hfl, err := ParseHosts(h.ReadFilePath)
	if err != nil {
		return nil, err
	}

	h.hostFileLines = hfl

	return h, nil
}

// Save rendered hosts file
func (h *Hosts) Save() error {
	return h.SaveAs(h.WriteFilePath)
}

// SaveAs saves rendered hosts file to the filename specified
func (h *Hosts) SaveAs(fileName string) error {
	if h.RawText != nil {
		return errors.New("cannot call Save or SaveAs with RawText. Use RenderHostsFile to return a string")
	}
	hfData := []byte(h.RenderHostsFile())

	h.Lock()
	defer h.Unlock()

	err := os.WriteFile(fileName, hfData, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Reload hosts file
func (h *Hosts) Reload() error {
	if h.RawText != nil {
		return errors.New("cannot call Reload with RawText")
	}
	h.Lock()
	defer h.Unlock()

	hfl, err := ParseHosts(h.ReadFilePath)
	if err != nil {
		return err
	}

	h.hostFileLines = hfl

	return nil
}

// RemoveAddresses removes all entries (lines) with the provided address.
func (h *Hosts) RemoveAddresses(addresses []string) {
	for _, address := range addresses {
		if h.RemoveFirstAddress(address) {
			h.RemoveAddress(address)
		}
	}
}

// RemoveAddress removes all entries (lines) with the provided address.
func (h *Hosts) RemoveAddress(address string) {
	if h.RemoveFirstAddress(address) {
		h.RemoveAddress(address)
	}
}

// RemoveFirstAddress removes the first entry (line) found with the provided address.
func (h *Hosts) RemoveFirstAddress(address string) bool {
	h.Lock()
	defer h.Unlock()

	for hflIdx := range h.hostFileLines {
		if address == h.hostFileLines[hflIdx].Address {
			h.hostFileLines = removeHFLElement(h.hostFileLines, hflIdx)
			return true
		}
	}

	return false
}

// RemoveCIDRs Remove CIDR Range (Classless inter-domain routing)
// examples:
//
//	127.1.0.0/16  = 127.1.0.0  -> 127.1.255.255
//	127.1.27.0/24 = 127.1.27.0 -> 127.1.27.255
func (h *Hosts) RemoveCIDRs(cidrs []string) error {
	addresses := make([]string, 0)

	// loop through all the CIDR ranges (we probably have less ranges than IPs)
	for _, cidr := range cidrs {

		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}

		hfLines := h.GetHostFileLines()

		for _, hfl := range *hfLines {
			ip := net.ParseIP(hfl.Address)
			if ip != nil {
				if ipnet.Contains(ip) {
					addresses = append(addresses, hfl.Address)
				}
			}
		}
	}

	h.RemoveAddresses(addresses)

	return nil
}

// RemoveHosts removes all hostname entries of the provided host slice
func (h *Hosts) RemoveHosts(hosts []string) {
	for _, host := range hosts {
		if h.RemoveFirstHost(host) {
			h.RemoveHost(host)
		}
	}
}

// RemoveHost removes all hostname entries of provided host
func (h *Hosts) RemoveHost(host string) {
	if h.RemoveFirstHost(host) {
		h.RemoveHost(host)
	}
}

// RemoveFirstHost the first hostname entry found and returns true if successful
func (h *Hosts) RemoveFirstHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	h.Lock()
	defer h.Unlock()

	for hflIdx := range h.hostFileLines {
		for hidx, hst := range h.hostFileLines[hflIdx].Hostnames {
			if hst == host {
				h.hostFileLines[hflIdx].Hostnames = removeStringElement(h.hostFileLines[hflIdx].Hostnames, hidx)

				// remove the address line if empty
				if len(h.hostFileLines[hflIdx].Hostnames) < 1 {
					h.hostFileLines = removeHFLElement(h.hostFileLines, hflIdx)
				}
				return true
			}
		}
	}

	return false
}

// AddHosts adds an array of hosts to the first matching address it finds
// or creates the address and adds the hosts
func (h *Hosts) AddHosts(address string, hosts []string) {
	for _, hst := range hosts {
		h.AddHost(address, hst)
	}
}

// AddHost adds a host to an address and removes the host
// from any existing address is may be associated with
func (h *Hosts) AddHost(addressRaw string, hostRaw string) {
	host := strings.TrimSpace(strings.ToLower(hostRaw))
	address := strings.TrimSpace(strings.ToLower(addressRaw))
	addressIP := net.ParseIP(address)
	if addressIP == nil {
		return
	}
	ipFamily := IPFamilyV4
	if addressIP.To4() == nil {
		ipFamily = IPFamilyV6
	}

	// does the host already exist
	if ok, exAdd, hflIdx := h.HostAddressLookup(host, ipFamily); ok {
		// if the address is the same we are done
		if address == exAdd {
			return
		}

		// if the hostname is at a different address, go and remove it from the address
		for hidx, hst := range h.hostFileLines[hflIdx].Hostnames {
			// for localhost, we can match more than one host
			if isLocalhost(address) {
				break
			}
			if hst == host {
				h.Lock()
				h.hostFileLines[hflIdx].Hostnames = removeStringElement(h.hostFileLines[hflIdx].Hostnames, hidx)
				h.Unlock()

				// remove the address line if empty
				if len(h.hostFileLines[hflIdx].Hostnames) < 1 {
					h.Lock()
					h.hostFileLines = removeHFLElement(h.hostFileLines, hflIdx)
					h.Unlock()
				}

				break // unless we should continue because it could have duplicates
			}
		}
	}

	// if the address exists add it to the address line
	for i, hfl := range h.hostFileLines {
		if hfl.Address == address {
			h.Lock()
			h.hostFileLines[i].Hostnames = append(h.hostFileLines[i].Hostnames, host)
			h.Unlock()
			return
		}
	}

	// the address and host do not already exist so go ahead and create them
	hfl := HostFileLine{
		LineType:  ADDRESS,
		Address:   address,
		Hostnames: []string{host},
	}

	h.Lock()
	h.hostFileLines = append(h.hostFileLines, hfl)
	h.Unlock()
}

// ListHostsByIP returns a list of hostnames associated with a given IP address
func (h *Hosts) ListHostsByIP(address string) []string {
	h.Lock()
	defer h.Unlock()

	var hosts []string

	for _, hsl := range h.hostFileLines {
		if hsl.Address == address {
			hosts = append(hosts, hsl.Hostnames...)
		}
	}

	return hosts
}

// ListAddressesByHost returns a list of IPs associated with a given hostname
func (h *Hosts) ListAddressesByHost(hostname string, exact bool) [][]string {
	h.Lock()
	defer h.Unlock()

	var addresses [][]string

	for _, hsl := range h.hostFileLines {
		for _, hst := range hsl.Hostnames {
			if hst == hostname {
				addresses = append(addresses, []string{hsl.Address, hst})
			}
			if exact == false && hst != hostname && strings.Contains(hst, hostname) {
				addresses = append(addresses, []string{hsl.Address, hst})
			}
		}
	}

	return addresses
}

// ListHostsByCIDR returns a list of IPs and hostnames associated with a given CIDR
func (h *Hosts) ListHostsByCIDR(cidr string) [][]string {
	h.Lock()
	defer h.Unlock()

	var ipHosts [][]string

	_, subnet, _ := net.ParseCIDR(cidr)
	for _, hsl := range h.hostFileLines {
		if subnet.Contains(net.ParseIP(hsl.Address)) {
			for _, hst := range hsl.Hostnames {
				ipHosts = append(ipHosts, []string{hsl.Address, hst})
			}
		}
	}

	return ipHosts
}

// HostAddressLookup returns true if the host is found, a string
// containing the address and the index of the hfl
func (h *Hosts) HostAddressLookup(host string, ipFamily IPFamily) (bool, string, int) {
	h.Lock()
	defer h.Unlock()

	host = strings.ToLower(strings.TrimSpace(host))

	for i, hfl := range h.hostFileLines {
		for _, hn := range hfl.Hostnames {
			ipAddr := net.ParseIP(hfl.Address)
			if ipAddr == nil || hn != host {
				continue
			}
			if ipFamily == IPFamilyV4 && ipAddr.To4() != nil {
				return true, hfl.Address, i
			}
			if ipFamily == IPFamilyV6 && ipAddr.To4() == nil {
				return true, hfl.Address, i
			}
		}
	}

	return false, "", 0
}

func (h *Hosts) RenderHostsFile() string {
	h.Lock()
	defer h.Unlock()

	hf := ""

	for _, hfl := range h.hostFileLines {
		hf = hf + fmt.Sprintln(lineFormatter(hfl))
	}

	return hf
}

func (h *Hosts) GetHostFileLines() *HostFileLines {
	h.Lock()
	defer h.Unlock()

	return &h.hostFileLines
}

func ParseHosts(path string) ([]HostFileLine, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseHostsFromString(string(input))
}

func ParseHostsFromString(input string) ([]HostFileLine, error) {
	inputNormalized := strings.Replace(input, "\r\n", "\n", -1)

	dataLines := strings.Split(inputNormalized, "\n")
	//remove extra blank line at end that does not exist in /etc/hosts file
	dataLines = dataLines[:len(dataLines)-1]

	hostFileLines := make([]HostFileLine, len(dataLines))

	// trim leading and trailing whitespace
	for i, l := range dataLines {
		curLine := &hostFileLines[i]
		curLine.OriginalLineNum = i
		curLine.Raw = l

		// trim line
		curLine.Trimmed = strings.TrimSpace(l)

		// check for comment
		if strings.HasPrefix(curLine.Trimmed, "#") {
			curLine.LineType = COMMENT
			continue
		}

		if curLine.Trimmed == "" {
			curLine.LineType = EMPTY
			continue
		}

		curLineSplit := strings.SplitN(curLine.Trimmed, "#", 2)
		if len(curLineSplit) > 1 {
			curLine.Comment = curLineSplit[1]
		}
		curLine.Trimmed = curLineSplit[0]

		curLine.Parts = strings.Fields(curLine.Trimmed)

		if len(curLine.Parts) > 1 {
			curLine.LineType = ADDRESS
			curLine.Address = strings.ToLower(curLine.Parts[0])
			// lower case all
			for _, p := range curLine.Parts[1:] {
				curLine.Hostnames = append(curLine.Hostnames, strings.ToLower(p))
			}

			continue
		}

		// if we can't figure out what this line is
		// at this point mark it as unknown
		curLine.LineType = UNKNOWN

	}

	return hostFileLines, nil
}

// removeStringElement removed an element of a string slice
func removeStringElement(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

// removeHFLElement removed an element of a HostFileLine slice
func removeHFLElement(slice []HostFileLine, s int) []HostFileLine {
	return append(slice[:s], slice[s+1:]...)
}

// lineFormatter
func lineFormatter(hfl HostFileLine) string {

	if hfl.LineType < ADDRESS {
		return hfl.Raw
	}

	if len(hfl.Comment) > 0 {
		return fmt.Sprintf("%-16s %s #%s", hfl.Address, strings.Join(hfl.Hostnames, " "), hfl.Comment)
	}
	return fmt.Sprintf("%-16s %s", hfl.Address, strings.Join(hfl.Hostnames, " "))
}

// IPLocalhost is a regex pattern for IPv4 or IPv6 loopback range.
const ipLocalhost = `((127\.([0-9]{1,3}\.){2}[0-9]{1,3})|(::1)$)`

var localhostIPRegexp = regexp.MustCompile(ipLocalhost)

func isLocalhost(address string) bool {
	return localhostIPRegexp.MatchString(address)
}
