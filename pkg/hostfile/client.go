package hostfile

import "github.com/txn2/txeh"

// Hostfile manages the hostname entries Monday maps in the system hosts file
type Hostfile interface {
	AddHost(ip, hostname string) error
	RemoveHost(hostname string) error
	HasHost(hostname string) bool
}

// hostfile represents the host file manager client
type hostfile struct {
	hosts *txeh.Hosts
}

// NewClient returns a new Hostfile manager client
func NewClient() (*hostfile, error) {
	hosts, err := txeh.NewHostsDefault()
	if err != nil {
		return nil, err
	}

	return &hostfile{
		hosts: hosts,
	}, nil
}

// AddHost adds a new host / ip entry into the hosts file
func (h *hostfile) AddHost(ip, hostname string) error {
	if err := h.hosts.Reload(); err != nil {
		return err
	}

	h.hosts.AddHost(ip, hostname)

	return h.hosts.Save()
}

// RemoveHost removes a given hostname from the hosts file
func (h *hostfile) RemoveHost(hostname string) error {
	if err := h.hosts.Reload(); err != nil {
		return err
	}

	h.hosts.RemoveHost(hostname)

	return h.hosts.Save()
}

// HasHost returns true when the given hostname is mapped in the hosts file
func (h *hostfile) HasHost(hostname string) bool {
	if err := h.hosts.Reload(); err != nil {
		return false
	}

	for _, family := range []txeh.IPFamily{txeh.IPFamilyV4, txeh.IPFamilyV6} {
		if found, _, _ := h.hosts.HostAddressLookup(hostname, family); found {
			return true
		}
	}

	return false
}
