// Copyright 2013-2015 go-diameter authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Diameter client.

package diam

import (
	"crypto/tls"
	"net"

	"github.com/ibrohimislam/go-diameter/diam/dict"
)

// Dial connects to the peer pointed to by addr and returns the Conn that
// can be used to send diameter messages. Incoming messages are handled
// by the handler, which is typically nil and DefaultServeMux is used.
// If dict is nil, dict.Default is used.
func Dial(addr string, handler Handler, dp *dict.Parser) (Conn, error) {
	srv := &Server{Addr: addr, Handler: handler, Dict: dp}
	return dial(srv)
}

func dial(srv *Server) (Conn, error) {
	addr := srv.Addr
	if len(addr) == 0 {
		addr = ":3868"
	}
	rw, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c, err := srv.newConn(rw)
	if err != nil {
		return nil, err
	}
	go c.serve()
	return c.writer, nil
}

// DialTLS is the same as Dial, but for TLS.
func DialTLS(addr, certFile, keyFile string, handler Handler, dp *dict.Parser) (Conn, error) {
	srv := &Server{Addr: addr, Handler: handler, Dict: dp}
	return dialTLS(srv, certFile, keyFile)
}

func dialTLS(srv *Server, certFile, keyFile string) (Conn, error) {
	addr := srv.Addr
	if len(addr) == 0 {
		addr = ":3868"
	}
	config := &tls.Config{InsecureSkipVerify: true}
	if srv.TLSConfig != nil {
		*config = *srv.TLSConfig
	}
	if len(certFile) != 0 {
		var err error
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}
	}
	rw, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c, err := srv.newConn(tls.Client(rw, config))
	if err != nil {
		return nil, err
	}
	go c.serve()
	return c.writer, nil
}
