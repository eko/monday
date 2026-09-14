package proxy

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoopbackCommand(t *testing.T) {
	testCases := []struct {
		name         string
		goos         string
		action       aliasAction
		available    map[string]bool
		expectedCmd  string
		expectedArgs []string
		expectedErr  error
	}{
		{
			name:         "darwin add",
			goos:         "darwin",
			action:       aliasAdd,
			expectedCmd:  "ifconfig",
			expectedArgs: []string{"lo0", "alias", "127.0.1.1", "up"},
		},
		{
			name:         "darwin remove",
			goos:         "darwin",
			action:       aliasRemove,
			expectedCmd:  "ifconfig",
			expectedArgs: []string{"lo0", "-alias", "127.0.1.1"},
		},
		{
			name:         "linux with iproute2 add",
			goos:         "linux",
			action:       aliasAdd,
			available:    map[string]bool{"ip": true, "ifconfig": true},
			expectedCmd:  "ip",
			expectedArgs: []string{"addr", "add", "127.0.1.1/32", "dev", "lo0"},
		},
		{
			name:         "linux with iproute2 remove",
			goos:         "linux",
			action:       aliasRemove,
			available:    map[string]bool{"ip": true},
			expectedCmd:  "ip",
			expectedArgs: []string{"addr", "del", "127.0.1.1/32", "dev", "lo0"},
		},
		{
			name:         "linux with ifconfig only add",
			goos:         "linux",
			action:       aliasAdd,
			available:    map[string]bool{"ifconfig": true},
			expectedCmd:  "ifconfig",
			expectedArgs: []string{"lo0", "127.0.1.1", "up"},
		},
		{
			name:        "linux with ifconfig only remove is unsupported",
			goos:        "linux",
			action:      aliasRemove,
			available:   map[string]bool{"ifconfig": true},
			expectedErr: ErrAliasRemovalUnsupported,
		},
		{
			name:        "linux without any tool",
			goos:        "linux",
			action:      aliasAdd,
			expectedErr: errors.New("neither 'ip' nor 'ifconfig' command is available to manage the loopback interface"),
		},
		{
			name:        "unsupported OS",
			goos:        "windows",
			action:      aliasAdd,
			expectedErr: errors.New("sorry, it seems your OS (windows) is not available yet"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lookPath = func(file string) (string, error) {
				if testCase.available[file] {
					return "/sbin/" + file, nil
				}

				return "", errors.New("not found")
			}
			t.Cleanup(func() { lookPath = defaultLookPath })

			command, args, err := loopbackCommand(testCase.goos, testCase.action, "lo0", "127.0.1.1")

			if testCase.expectedErr != nil {
				assert.EqualError(t, err, testCase.expectedErr.Error())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedCmd, command)
			assert.Equal(t, testCase.expectedArgs, args)
		})
	}
}

func TestIsMondayLoopbackIP(t *testing.T) {
	testCases := []struct {
		ip   string
		want bool
	}{
		{ip: "127.0.0.1", want: false},
		{ip: "127.0.0.2", want: false},
		{ip: "127.0.1.0", want: true},
		{ip: "127.0.1.1", want: true},
		{ip: "127.0.255.255", want: true},
		{ip: "127.1.0.0", want: true},
		{ip: "10.0.0.1", want: false},
		{ip: "::1", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.ip, func(t *testing.T) {
			assert.Equal(t, testCase.want, isMondayLoopbackIP(net.ParseIP(testCase.ip)))
		})
	}
}

func TestIsAlreadyAssigned(t *testing.T) {
	// Given: addresses as reported on macOS (/8) and on Linux with iproute2 (/32)
	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("127.0.1.1"), Mask: net.CIDRMask(8, 32)},
		&net.IPNet{IP: net.ParseIP("127.0.1.2"), Mask: net.CIDRMask(32, 32)},
		&net.IPAddr{IP: net.ParseIP("127.0.1.3")},
	}

	// When - Then
	assert.True(t, isAlreadyAssigned(net.IPv4(127, 0, 1, 1), addrs))
	assert.True(t, isAlreadyAssigned(net.IPv4(127, 0, 1, 2), addrs))
	assert.True(t, isAlreadyAssigned(net.IPv4(127, 0, 1, 3), addrs))
	assert.False(t, isAlreadyAssigned(net.IPv4(127, 0, 1, 4), addrs))
}
