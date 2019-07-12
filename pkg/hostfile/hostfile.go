package hostfile

import "github.com/txn2/txeh"

func initClient() (*txeh.Hosts, error) {
	hosts, err := txeh.NewHostsDefault()
	if err != nil {
		panic(err)
	}

	// Always reload to ensure we have the latest version
	hosts.Reload()

	return hosts, err
}

func AddHost(ip, hostname string) error {
	client, err := initClient()
	if err != nil {
		return err
	}

	client.AddHost(ip, hostname)
	err = client.Save()
	if err != nil {
		return err
	}

	return nil
}

func RemoveHost(hostname string) error {
	client, err := initClient()
	if err != nil {
		return err
	}

	client.RemoveHost(hostname)
	client.Save()

	return nil
}
