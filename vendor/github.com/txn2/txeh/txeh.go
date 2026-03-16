// Package txeh provides /etc/hosts file management capabilities.
// It offers a simple interface for adding, removing, and querying hostname-to-IP mappings
// with thread-safe operations and cross-platform support (Linux, macOS, Windows).
package txeh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// Line type constants for HostFileLine.
const (
	UNKNOWN = 0  // Unknown line type.
	EMPTY   = 10 // Empty line.
	COMMENT = 20 // Comment line starting with #.
	ADDRESS = 30 // Address line with IP and hostnames.
)

// DefaultMaxHostsPerLineWindows is the default maximum number of hostnames per line on Windows.
// Windows has a limitation where lines with more than ~9 hostnames may not resolve correctly.
const DefaultMaxHostsPerLineWindows = 9

// osWindows is the runtime.GOOS value for Windows.
const osWindows = "windows"

// IPFamily represents the IP address family (IPv4 or IPv6).
type IPFamily int64

// IP address family constants.
const (
	IPFamilyV4 IPFamily = iota // IPv4 address family.
	IPFamilyV6                 // IPv6 address family.
)

// HostsConfig contains configuration for reading and writing hosts files.
type HostsConfig struct {
	ReadFilePath  string
	WriteFilePath string
	// RawText for input. If RawText is set ReadFilePath, WriteFilePath are ignored. Use RenderHostsFile rather
	// than save to get the results.
	RawText *string
	// MaxHostsPerLine limits the number of hostnames per line when adding hosts.
	// This is useful for Windows which has a limitation of ~9 hostnames per line.
	// Values:
	//   0  = auto-detect (Windows: 9, others: unlimited)
	//  -1  = force unlimited (no limit)
	//  >0  = explicit limit
	MaxHostsPerLine int
	// AutoFlush triggers a DNS cache flush after every successful Save/SaveAs.
	// Flush failures are returned as *FlushError, distinguishable via errors.As.
	AutoFlush bool
}

// Hosts represents a parsed hosts file with thread-safe operations.
type Hosts struct {
	mu sync.Mutex
	*HostsConfig
	hostFileLines HostFileLines
}

// AddressLocations maps an address to its location in the HFL.
type AddressLocations map[string]int

// HostLocations maps a hostname to an original line number.
type HostLocations map[string]int

// HostFileLines is a slice of HostFileLine entries.
type HostFileLines []HostFileLine

// HostFileLine represents a single line in a hosts file.
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

// NewHostsDefault returns a hosts object with default configuration.
func NewHostsDefault() (*Hosts, error) {
	return NewHosts(&HostsConfig{})
}

// NewHosts returns a new hosts object.
func NewHosts(hc *HostsConfig) (*Hosts, error) {
	h := &Hosts{HostsConfig: hc}
	h.mu.Lock()
	defer h.mu.Unlock()

	defaultHostsFile := "/etc/hosts"

	if runtime.GOOS == osWindows {
		defaultHostsFile = winDefaultHostsFile()
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

// Save renders and writes the hosts file to the configured write path.
func (h *Hosts) Save() error {
	return h.SaveAs(h.WriteFilePath)
}

// SaveAs saves rendered hosts file to the filename specified.
func (h *Hosts) SaveAs(fileName string) error {
	if h.RawText != nil {
		return errors.New("cannot call Save or SaveAs with RawText. Use RenderHostsFile to return a string")
	}
	hfData := []byte(h.RenderHostsFile())

	h.mu.Lock()
	defer h.mu.Unlock()

	err := os.WriteFile(filepath.Clean(fileName), hfData, 0o644) // #nosec G306 -- hosts file must be world-readable (0644) for DNS resolution
	if err != nil {
		return fmt.Errorf("write hosts file %s: %w", fileName, err)
	}

	if h.AutoFlush {
		if flushErr := FlushDNSCache(); flushErr != nil {
			return flushErr
		}
	}

	return nil
}

// Reload re-reads the hosts file from disk and replaces the in-memory state.
// This is part of the public API for consumers who manage long-lived Hosts instances.
func (h *Hosts) Reload() error {
	if h.RawText != nil {
		return errors.New("cannot call Reload with RawText")
	}
	h.mu.Lock()
	defer h.mu.Unlock()

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
	h.mu.Lock()
	defer h.mu.Unlock()

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
			return fmt.Errorf("parse CIDR %s: %w", cidr, err)
		}

		hfLines := h.GetHostFileLines()

		for _, hfl := range hfLines {
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

// RemoveHosts removes all hostname entries of the provided host slice.
func (h *Hosts) RemoveHosts(hosts []string) {
	for _, host := range hosts {
		if h.RemoveFirstHost(host) {
			h.RemoveHost(host)
		}
	}
}

// RemoveHost removes all hostname entries of provided host.
func (h *Hosts) RemoveHost(host string) {
	if h.RemoveFirstHost(host) {
		h.RemoveHost(host)
	}
}

// RemoveFirstHost removes the first hostname entry found and returns true if successful.
func (h *Hosts) RemoveFirstHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	h.mu.Lock()
	defer h.mu.Unlock()

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

// RemoveByComments removes all host entries that have any of the specified comments.
// This removes entire lines where the comment matches.
// This is part of the public API for bulk comment-based cleanup (e.g., kubefwd teardown).
func (h *Hosts) RemoveByComments(comments []string) {
	for _, comment := range comments {
		h.RemoveByComment(comment)
	}
}

// RemoveByComment removes all host entries that have the specified comment.
// This removes entire lines where the comment matches.
func (h *Hosts) RemoveByComment(comment string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	comment = strings.TrimSpace(comment)
	var newLines HostFileLines

	for _, hfl := range h.hostFileLines {
		if hfl.Comment != comment {
			newLines = append(newLines, hfl)
		}
	}

	h.hostFileLines = newLines
}

// AddHosts adds an array of hosts to the first matching address it finds
// or creates the address and adds the hosts.
func (h *Hosts) AddHosts(address string, hosts []string) {
	for _, hst := range hosts {
		h.AddHost(address, hst)
	}
}

// AddHostsWithComment adds an array of hosts to an address with a comment.
// All hosts will share the same comment. If the address already exists with
// the same comment, hosts are appended to that line (respecting MaxHostsPerLine).
// If the address exists with a different comment, a new line is created.
func (h *Hosts) AddHostsWithComment(address string, hosts []string, comment string) {
	for _, hst := range hosts {
		h.AddHostWithComment(address, hst, comment)
	}
}

// AddHost adds a host to an address and removes the host
// from any existing address it may be associated with.
func (h *Hosts) AddHost(addressRaw, hostRaw string) {
	h.addHostWithComment(addressRaw, hostRaw, "")
}

// AddHostWithComment adds a host to an address with an inline comment.
// The comment will appear after the hostnames on the line (e.g., "127.0.0.1 host # comment").
// If the address already exists with the same comment, the host is appended to that line
// (respecting MaxHostsPerLine). If the address exists with a different comment, a new line
// is created with the specified comment.
func (h *Hosts) AddHostWithComment(addressRaw, hostRaw, comment string) {
	h.addHostWithComment(addressRaw, hostRaw, comment)
}

// addHostWithComment is the internal implementation that handles both
// commented and non-commented host additions.
func (h *Hosts) addHostWithComment(addressRaw, hostRaw, comment string) { //nolint:revive // internal method mirrors public API naming
	host := strings.TrimSpace(strings.ToLower(hostRaw))
	address := strings.TrimSpace(strings.ToLower(addressRaw))
	// Normalize comment: trim spaces, but don't add/remove the # prefix
	// (the # is handled during rendering)
	comment = strings.TrimSpace(comment)

	addressIP := net.ParseIP(address)
	if addressIP == nil {
		return
	}
	ipFamily := IPFamilyV4
	if addressIP.To4() == nil {
		ipFamily = IPFamilyV6
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// does the host already exist
	ok, exAdd, hflIdx := h.hostAddressLookupLocked(host, ipFamily)
	if ok {
		if address == exAdd {
			return // already at correct address
		}
		// hostname is at a different address, remove it from there
		h.removeHostFromLineLocked(hflIdx, host, address)
	}

	// Get the effective max hosts per line limit
	maxPerLine := h.getEffectiveMaxHostsPerLine()

	// if the address exists with matching comment, add it to that line if there's room
	for i, hfl := range h.hostFileLines {
		if hfl.Address == address && hfl.Comment == comment {
			// Check if this line has room (0 means unlimited)
			if maxPerLine <= 0 || len(h.hostFileLines[i].Hostnames) < maxPerLine {
				h.hostFileLines[i].Hostnames = append(h.hostFileLines[i].Hostnames, host)
				return
			}
			// This line is full, continue looking for another line with the same address and comment
		}
	}

	// No existing line with matching address and comment found (or all are full), create a new line
	hfl := HostFileLine{
		LineType:  ADDRESS,
		Address:   address,
		Hostnames: []string{host},
		Comment:   comment,
	}

	h.hostFileLines = append(h.hostFileLines, hfl)
}

// ListHostsByIP returns a list of hostnames associated with a given IP address.
func (h *Hosts) ListHostsByIP(address string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	var hosts []string

	for _, hsl := range h.hostFileLines {
		if hsl.Address == address {
			hosts = append(hosts, hsl.Hostnames...)
		}
	}

	return hosts
}

// ListAddressesByHost returns a list of IPs associated with a given hostname.
func (h *Hosts) ListAddressesByHost(hostname string, exact bool) [][]string {
	h.mu.Lock()
	defer h.mu.Unlock()

	var addresses [][]string

	for _, hsl := range h.hostFileLines {
		for _, hst := range hsl.Hostnames {
			if hst == hostname {
				addresses = append(addresses, []string{hsl.Address, hst})
			}
			if !exact && hst != hostname && strings.Contains(hst, hostname) {
				addresses = append(addresses, []string{hsl.Address, hst})
			}
		}
	}

	return addresses
}

// ListHostsByCIDR returns a list of IPs and hostnames associated with a given CIDR.
func (h *Hosts) ListHostsByCIDR(cidr string) [][]string {
	h.mu.Lock()
	defer h.mu.Unlock()

	var ipHosts [][]string

	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ipHosts
	}

	for _, hsl := range h.hostFileLines {
		if subnet.Contains(net.ParseIP(hsl.Address)) {
			for _, hst := range hsl.Hostnames {
				ipHosts = append(ipHosts, []string{hsl.Address, hst})
			}
		}
	}

	return ipHosts
}

// ListHostsByComment returns all hostnames on lines with the given comment.
func (h *Hosts) ListHostsByComment(comment string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	comment = strings.TrimSpace(comment)
	var hosts []string

	for _, hsl := range h.hostFileLines {
		if hsl.Comment == comment {
			hosts = append(hosts, hsl.Hostnames...)
		}
	}

	return hosts
}

// HostAddressLookup returns true if the host is found, the address string,
// and the index of the host file line. This is part of the public API for
// consumers that need direct address lookups by IP family.
func (h *Hosts) HostAddressLookup(host string, ipFamily IPFamily) (found bool, address string, idx int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.hostAddressLookupLocked(host, ipFamily)
}

// hostAddressLookupLocked is the internal version that assumes the lock is already held.
func (h *Hosts) hostAddressLookupLocked(host string, ipFamily IPFamily) (found bool, address string, idx int) {
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

// RenderHostsFile returns the hosts file content as a formatted string.
func (h *Hosts) RenderHostsFile() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	var sb strings.Builder
	for _, hfl := range h.hostFileLines {
		sb.WriteString(lineFormatter(hfl))
		sb.WriteByte('\n')
	}

	return sb.String()
}

// GetHostFileLines returns a copy of all parsed host file lines.
func (h *Hosts) GetHostFileLines() HostFileLines {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Return a copy to prevent external modification of internal state
	result := make(HostFileLines, len(h.hostFileLines))
	copy(result, h.hostFileLines)
	return result
}

// getEffectiveMaxHostsPerLine returns the effective maximum hosts per line.
// Returns 0 for unlimited.
func (h *Hosts) getEffectiveMaxHostsPerLine() int {
	if h.HostsConfig == nil {
		// No config, use auto-detect
		if runtime.GOOS == osWindows {
			return DefaultMaxHostsPerLineWindows
		}
		return 0
	}

	if h.MaxHostsPerLine > 0 {
		// Explicit limit set
		return h.MaxHostsPerLine
	}

	if h.MaxHostsPerLine < 0 {
		// Explicitly unlimited
		return 0
	}

	// MaxHostsPerLine == 0: auto-detect based on OS
	if runtime.GOOS == osWindows {
		return DefaultMaxHostsPerLineWindows
	}
	return 0
}

// ParseHosts reads and parses a hosts file from the given path.
func ParseHosts(path string) ([]HostFileLine, error) {
	input, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read hosts file %s: %w", path, err)
	}
	return ParseHostsFromString(string(input))
}

// ParseHostsFromString parses hosts file content from a string.
func ParseHostsFromString(input string) ([]HostFileLine, error) {
	inputNormalized := strings.ReplaceAll(input, "\r\n", "\n")

	dataLines := strings.Split(inputNormalized, "\n")
	// remove extra blank line at end that does not exist in /etc/hosts file
	// only if the file ended with a newline (last element is empty)
	if len(dataLines) > 0 && dataLines[len(dataLines)-1] == "" {
		dataLines = dataLines[:len(dataLines)-1]
	}

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
			curLine.Comment = strings.TrimSpace(curLineSplit[1])
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

// removeStringElement removes an element of a string slice.
func removeStringElement(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

// removeHFLElement removes an element of a HostFileLine slice.
func removeHFLElement(slice []HostFileLine, s int) []HostFileLine {
	return append(slice[:s], slice[s+1:]...)
}

// removeHostFromLineLocked removes a hostname from a specific line and cleans up empty lines.
// Must be called with lock held. For localhost addresses, this is a no-op since
// the same hostname can exist at multiple localhost addresses.
func (h *Hosts) removeHostFromLineLocked(hflIdx int, host, newAddress string) {
	if isLocalhost(newAddress) {
		return
	}

	for hidx, hst := range h.hostFileLines[hflIdx].Hostnames {
		if hst != host {
			continue
		}
		h.hostFileLines[hflIdx].Hostnames = removeStringElement(h.hostFileLines[hflIdx].Hostnames, hidx)
		if len(h.hostFileLines[hflIdx].Hostnames) == 0 {
			h.hostFileLines = removeHFLElement(h.hostFileLines, hflIdx)
		}
		return
	}
}

// lineFormatter formats a single host file line as a string.
func lineFormatter(hfl HostFileLine) string {
	if hfl.LineType < ADDRESS {
		return hfl.Raw
	}

	if hfl.Comment != "" {
		return fmt.Sprintf("%-16s %s # %s", hfl.Address, strings.Join(hfl.Hostnames, " "), hfl.Comment)
	}
	return fmt.Sprintf("%-16s %s", hfl.Address, strings.Join(hfl.Hostnames, " "))
}

// ipLocalhost is a regex pattern for IPv4 or IPv6 loopback range.
const ipLocalhost = `((127\.(\d{1,3}\.){2}\d{1,3})|(::1)$)`

var localhostIPRegexp = regexp.MustCompile(ipLocalhost)

func isLocalhost(address string) bool {
	return localhostIPRegexp.MatchString(address)
}

// winDefaultHostsFile returns the default hosts file path for Windows.
// It tries to use the SystemRoot environment variable. If that is not set,
// it falls back to C:\Windows\System32\Drivers\etc\hosts.
func winDefaultHostsFile() string {
	if r := os.Getenv("SystemRoot"); r != "" {
		return filepath.Join(r, "System32", "drivers", "etc", "hosts")
	}

	// fallback to C:\
	return `C:\Windows\System32\drivers\etc\hosts`
}
