// package shortcut constructs a dialer with IPv4 and IPv6 subnets. When the
// address to dial belongs to one of the subnets, it dial via the direct
// dialer, i.e., the shortcut, and dial the proxiedDialer otherwise.

package shortcut

import (
	"bufio"
	"context"
	"io"
	"net"
)

type Shortcut interface {
	// Allow checks if the address is allowed to use shortcut and returns true
	// together with the resolved IP address if so.
	Allow(ctx context.Context, addr string) (bool, net.IP)
}

type shortcut struct {
	v4list   *sortList
	v6list   *sortList
	resolver *net.Resolver
}

// NewFromReader is a helper to create shortcut from readers. The content
// should be in CIDR format, one entry per line.
func NewFromReader(v4 io.Reader, v6 io.Reader) Shortcut {
	return New(readLines(v4), readLines(v6))
}

func readLines(r io.Reader) []string {
	lines := []string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// New creates a new shortcut from the subnets.
func New(ipv4Subnets []string, ipv6Subnets []string) Shortcut {
	log.Debugf("Creating shortcut with %d ipv4 subnets and %d ipv6 subnets",
		len(ipv4Subnets),
		len(ipv6Subnets),
	)
	return &shortcut{
		v4list: newSortList(ipv4Subnets),
		v6list: newSortList(ipv6Subnets),
		// Prefers the system resolver by default, hopefully can use OS DNS cache.
		resolver: net.DefaultResolver,
	}
}

func (s *shortcut) Allow(ctx context.Context, addr string) (bool, net.IP) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	addrs, err := s.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false, nil
	}
	for _, addr := range addrs {
		ip := addr.IP.To4()
		if ip != nil {
			return s.v4list.Contains(ip), ip
		}
		if ip = ip.To16(); ip != nil {
			return s.v6list.Contains(ip), ip
		}
	}
	return false, nil
}
