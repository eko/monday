package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProxy(t *testing.T) {
	// When
	proxy := NewProxy()

	// Then
	assert.IsType(t, new(Proxy), proxy)

	assert.Len(t, proxy.ProxyForwards, 0)
	assert.Len(t, proxy.attributedIPs, 0)
}
