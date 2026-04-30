package main

import (
	"context"
	"net"
	"strings"
	"time"
)

func initCustomResolver() {
	server := strings.TrimSpace(customDNSServer)
	if server == "" {
		customResolver = nil
		return
	}
	if host, port, err := net.SplitHostPort(server); err == nil {
		if host == "" || port == "" {
			server = net.JoinHostPort(strings.Trim(host, "[]"), "53")
		}
	} else if ip := net.ParseIP(strings.Trim(server, "[]")); ip != nil {
		server = net.JoinHostPort(ip.String(), "53")
	} else {
		server = net.JoinHostPort(server, "53")
	}
	customResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: 5 * time.Second}
			conn, err := dialer.DialContext(ctx, network, server)
			if err == nil || network == "tcp" {
				return conn, err
			}
			return dialer.DialContext(ctx, "tcp", server)
		},
	}
}

func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 30 * time.Second}
	if customResolver != nil {
		dialer.Resolver = customResolver
	}
	return dialer.DialContext(ctx, network, addr)
}
