package hostfile

import "github.com/txn2/txeh"

type HostfileInterface interface {
	AddHost(ip, hostname string) error
	RemoveHost(hostname string) error
}

// Hostfile represents the host file manager client
type Hostfile struct {
	hosts *txeh.Hosts
}

// NewClient returns a new Hostfile manager client
func NewClient() (*Hostfile, error) {
	hosts, err := txeh.NewHostsDefault()
	if err != nil {
		panic(err)
	}

	return &Hostfile{
		hosts: hosts,
	}, err
}

// AddHost adds a new host / ip entry into the hosts file
func (h *Hostfile) AddHost(ip, hostname string) error {
	h.hosts.Reload()
	h.hosts.AddHost(ip, hostname)
	err := h.hosts.Save()
	if err != nil {
		return err
	}

	return nil
}

// RemoveHost removes a given hostname from the hosts file
func (h *Hostfile) RemoveHost(hostname string) error {
	h.hosts.Reload()
	h.hosts.RemoveHost(hostname)
	h.hosts.Save()

	return nil
}
