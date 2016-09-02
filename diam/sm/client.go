// Copyright 2013-2015 go-diameter authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package sm

import (
	"errors"
	"net"
	"time"

	"github.com/ibrohimislam/go-diameter/diam"
	"github.com/ibrohimislam/go-diameter/diam/avp"
	"github.com/ibrohimislam/go-diameter/diam/datatype"
	"github.com/ibrohimislam/go-diameter/diam/dict"
	"github.com/ibrohimislam/go-diameter/diam/sm/smparser"
)

var (
	// ErrMissingStateMachine is returned by Dial or DialTLS when
	// the Client does not have a valid StateMachine set.
	ErrMissingStateMachine = errors.New("client state machine is nil")

	// ErrHandshakeTimeout is returned by Dial or DialTLS when the
	// client does not receive a handshake answer from the server.
	//
	// If the client is configured to retransmit messages, the
	// handshake timeout only occurs after all retransmits are
	// attempted and none has an aswer.
	ErrHandshakeTimeout = errors.New("handshake timeout (no response)")
)

// A Client is a diameter client that automatically performs a handshake
// with the server after the connection is established.
//
// It sends a Capabilities-Exchange-Request with the AVPs defined in it,
// and expects a Capabilities-Exchange-Answer with a success (2001) result
// code. If enabled, the client will send Device-Watchdog-Request messages
// in background until the connection is terminated.
//
// By default, retransmission and watchdog are disabled. Retransmission is
// enabled by setting MaxRetransmits to a number greater than zero, and
// watchdog is enabled by setting EnableWatchdog to true.
type Client struct {
	Dict                        *dict.Parser  // Dictionary parser (uses dict.Default if unset)
	Handler                     *StateMachine // Message handler
	MaxRetransmits              uint          // Max number of retransmissions before aborting
	RetransmitInterval          time.Duration // Interval between retransmissions (default 1s)
	EnableWatchdog              bool          // Enable automatic DWR
	WatchdogInterval            time.Duration // Interval between DWRs (default 5s)
	SupportedVendorID           []*diam.AVP   // Supported vendor ID
	AcctApplicationID           []*diam.AVP   // Acct applications
	AuthApplicationID           []*diam.AVP   // Auth applications
	VendorSpecificApplicationID []*diam.AVP   // Vendor specific applications
}

// Dial calls the address set as ip:port, performs a handshake and optionally
// start a watchdog goroutine in background.
func (cli *Client) Dial(addr string) (diam.Conn, error) {
	return cli.dial(func() (diam.Conn, error) {
		return diam.Dial(addr, cli.Handler, cli.Dict)
	})
}

// DialTLS is like Dial, but using TLS.
func (cli *Client) DialTLS(addr, certFile, keyFile string) (diam.Conn, error) {
	return cli.dial(func() (diam.Conn, error) {
		return diam.DialTLS(addr, certFile, keyFile, cli.Handler, cli.Dict)
	})
}

type dialFunc func() (diam.Conn, error)

func (cli *Client) dial(f dialFunc) (diam.Conn, error) {
	if err := cli.validate(); err != nil {
		return nil, err
	}
	c, err := f()
	if err != nil {
		return nil, err
	}
	return cli.handshake(c)
}

func (cli *Client) validate() error {
	if cli.Handler == nil {
		return ErrMissingStateMachine
	}
	if cli.Dict == nil {
		cli.Dict = dict.Default
	}
	if cli.RetransmitInterval == 0 {
		// Set default RetransmitInterval.
		cli.RetransmitInterval = time.Second
	}
	if cli.WatchdogInterval == 0 {
		// Set default WatchdogInterval
		cli.WatchdogInterval = 5 * time.Second
	}
	app := &smparser.Application{
		AcctApplicationID:           cli.AcctApplicationID,
		AuthApplicationID:           cli.AuthApplicationID,
		VendorSpecificApplicationID: cli.VendorSpecificApplicationID,
	}
	// Make sure the given applications exist in the dictionary
	// before sending a CER.
	_, err := app.Parse(cli.Dict)
	if err != nil {
		return err
	}
	return nil
}

func (cli *Client) handshake(c diam.Conn) (diam.Conn, error) {
	ip, _, err := net.SplitHostPort(c.LocalAddr().String())
	if err != nil {
		return nil, err
	}
	m := cli.makeCER(net.ParseIP(ip))
	// Ignore CER, but not DWR.
	cli.Handler.mux.HandleFunc("CER", func(c diam.Conn, m *diam.Message) {})
	// Handle CEA and DWA.
	errc := make(chan error)
	dwac := make(chan struct{})
	cli.Handler.mux.Handle("CEA", handleCEA(cli.Handler, errc))
	cli.Handler.mux.Handle("DWA", handshakeOK(handleDWA(cli.Handler, dwac)))
	for i := 0; i < (int(cli.MaxRetransmits) + 1); i++ {
		_, err := m.WriteTo(c)
		if err != nil {
			return nil, err
		}
		select {
		case err := <-errc: // Wait for CEA.
			if err != nil {
				close(errc)
				return nil, err
			}
			if cli.EnableWatchdog {
				go cli.watchdog(c, dwac)
			}
			return c, nil
		case <-time.After(cli.RetransmitInterval):
		}
	}
	c.Close()
	return nil, ErrHandshakeTimeout
}

func (cli *Client) makeCER(ip net.IP) *diam.Message {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, cli.Dict)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, cli.Handler.cfg.OriginHost)
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, cli.Handler.cfg.OriginRealm)
	m.NewAVP(avp.HostIPAddress, avp.Mbit, 0, datatype.Address(ip))
	m.NewAVP(avp.VendorID, avp.Mbit, 0, cli.Handler.cfg.VendorID)
	m.NewAVP(avp.ProductName, 0, 0, cli.Handler.cfg.ProductName)
	if cli.Handler.cfg.OriginStateID != 0 {
		stateid := datatype.Unsigned32(cli.Handler.cfg.OriginStateID)
		m.NewAVP(avp.OriginStateID, avp.Mbit, 0, stateid)
	}
	if cli.SupportedVendorID != nil {
		for _, a := range cli.SupportedVendorID {
			m.AddAVP(a)
		}
	}
	if cli.AuthApplicationID != nil {
		for _, a := range cli.AuthApplicationID {
			m.AddAVP(a)
		}
	}
	m.NewAVP(avp.InbandSecurityID, avp.Mbit, 0, datatype.Unsigned32(0))
	if cli.AcctApplicationID != nil {
		for _, a := range cli.AcctApplicationID {
			m.AddAVP(a)
		}
	}
	if cli.VendorSpecificApplicationID != nil {
		for _, a := range cli.VendorSpecificApplicationID {
			m.AddAVP(a)
		}
	}
	if cli.Handler.cfg.FirmwareRevision != 0 {
		m.NewAVP(avp.FirmwareRevision, avp.Mbit, 0, cli.Handler.cfg.FirmwareRevision)
	}
	return m
}

func (cli *Client) watchdog(c diam.Conn, dwac chan struct{}) {
	disconnect := c.(diam.CloseNotifier).CloseNotify()
	var osid uint32 = uint32(cli.Handler.cfg.OriginStateID)
	for {
		select {
		case <-disconnect:
			return
		case <-time.After(cli.WatchdogInterval):
			cli.dwr(c, osid, dwac)
		}
	}
}

func (cli *Client) dwr(c diam.Conn, osid uint32, dwac chan struct{}) {
	m := cli.makeDWR(osid)
	for i := 0; i < (int(cli.MaxRetransmits) + 1); i++ {
		_, err := m.WriteTo(c)
		if err != nil {
			return
		}
		select {
		case <-dwac:
			return
		case <-time.After(cli.RetransmitInterval):
		}
	}
	// Watchdog failed, disconnect.
	c.Close()
}

func (cli *Client) makeDWR(osid uint32) *diam.Message {
	m := diam.NewRequest(diam.DeviceWatchdog, 0, cli.Dict)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, cli.Handler.cfg.OriginHost)
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, cli.Handler.cfg.OriginRealm)
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(osid))
	return m
}
