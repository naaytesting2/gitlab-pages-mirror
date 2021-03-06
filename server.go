package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/context"
	proxyproto "github.com/pires/go-proxyproto"
	"golang.org/x/net/http2"

	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
)

type keepAliveListener struct {
	net.Listener
}

type keepAliveSetter interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

func (ln *keepAliveListener) Accept() (net.Conn, error) {
	conn, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}

	kc := conn.(keepAliveSetter)
	kc.SetKeepAlive(true)
	kc.SetKeepAlivePeriod(3 * time.Minute)

	return conn, nil
}

func listenAndServe(fd uintptr, handler http.Handler, useHTTP2 bool, tlsConfig *tls.Config, limiter *netutil.Limiter, proxyv2 bool) error {
	// create server
	server := &http.Server{Handler: context.ClearHandler(handler), TLSConfig: tlsConfig}

	if useHTTP2 {
		err := http2.ConfigureServer(server, &http2.Server{})
		if err != nil {
			return err
		}
	}

	l, err := net.FileListener(os.NewFile(fd, "[socket]"))
	if err != nil {
		return fmt.Errorf("failed to listen on FD %d: %v", fd, err)
	}

	if limiter != nil {
		l = netutil.SharedLimitListener(l, limiter)
	}

	l = &keepAliveListener{l}

	if proxyv2 {
		l = &proxyproto.Listener{
			Listener: l,
			Policy: func(upstream net.Addr) (proxyproto.Policy, error) {
				return proxyproto.REQUIRE, nil
			},
		}
	}

	if tlsConfig != nil {
		l = tls.NewListener(l, server.TLSConfig)
	}

	return server.Serve(l)
}
