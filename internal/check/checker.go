package check

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
)

type FailureType string

const (
	FailureNone    FailureType = ""
	FailureHTTP    FailureType = "http"
	FailureNetwork FailureType = "network"
	FailureTimeout FailureType = "timeout"
)

type Result struct {
	StatusCode  int
	Latency     time.Duration
	Success     bool
	FailureType FailureType
	Error       error
}

type Checker struct {
	client      *http.Client
	dialContext func(context.Context, string, string) (net.Conn, error)
}

func NewChecker(client *http.Client) *Checker {
	if client == nil {
		client = &http.Client{}
	}
	clientCopy := *client
	priorRedirect := clientCopy.CheckRedirect
	clientCopy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if priorRedirect != nil {
			if err := priorRedirect(req, via); err != nil {
				return err
			}
		}
		return http.ErrUseLastResponse
	}
	return &Checker{
		client:      &clientCopy,
		dialContext: safeDialContext,
	}
}

func (c *Checker) Check(ctx context.Context, m *monitor.Monitor) Result {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, m.Timeout)
	defer cancel()

	parsedURL, err := url.ParseRequestURI(m.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Hostname() == "" || parsedURL.User != nil {
		return Result{Latency: time.Since(start), FailureType: FailureNetwork, Error: errors.New("URL must be an absolute HTTP or HTTPS URL without credentials")}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		m.URL,
		nil,
	)
	if err != nil {
		return Result{
			Latency:     time.Since(start),
			FailureType: FailureNetwork,
			Error:       err,
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = c.dialContext
	client := *c.client
	client.Transport = transport
	response, err := client.Do(req)
	if err != nil {
		failureType := FailureNetwork

		if errors.Is(err, context.DeadlineExceeded) {
			failureType = FailureTimeout
		}

		return Result{
			Latency:     time.Since(start),
			FailureType: failureType,
			Error:       err,
		}
	}

	defer response.Body.Close()

	success := response.StatusCode == m.ExpectedStatus

	result := Result{
		StatusCode: response.StatusCode,
		Latency:    time.Since(start),
		Success:    success,
	}

	if !success {
		result.FailureType = FailureHTTP
	}

	return result
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for target host")
	}
	for _, ip := range ips {
		if !isPublicIP(ip.IP) {
			return nil, fmt.Errorf("target resolves to a non-public IP address")
		}
	}
	dialer := net.Dialer{}
	var lastErr error
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func isPublicIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		if !v4.IsGlobalUnicast() || v4.IsPrivate() || v4.IsLoopback() || v4.IsLinkLocalUnicast() {
			return false
		}
		for _, block := range []string{"100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4"} {
			_, prefix, _ := net.ParseCIDR(block)
			if prefix.Contains(v4) {
				return false
			}
		}
		return true
	}
	// Only globally routable IPv6 unicast space is eligible; reject protocol
	// assignment and documentation ranges within that space as well.
	_, globalIPv6, _ := net.ParseCIDR("2000::/3")
	if !globalIPv6.Contains(ip) {
		return false
	}
	for _, block := range []string{"2001::/23", "2001:db8::/32"} {
		_, prefix, _ := net.ParseCIDR(block)
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}
