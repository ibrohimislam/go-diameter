// Copyright 2013-2015 go-diameter authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package sm

import (
	"fmt"
	"net"

	"github.com/ibrohimislam/go-diameter/diam"
	"github.com/ibrohimislam/go-diameter/diam/avp"
	"github.com/ibrohimislam/go-diameter/diam/datatype"
	"github.com/ibrohimislam/go-diameter/diam/sm/smparser"
	"github.com/ibrohimislam/go-diameter/diam/sm/smpeer"
)

// handleCER handles Capabilities-Exchange-Request messages.
//
// If mandatory AVPs such as Origin-Host or Origin-Realm
// are missing, we close the connection.
//
// See RFC 6733 section 5.3 for details.
func handleCER(sm *StateMachine) diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		ctx := c.Context()
		if _, ok := smpeer.FromContext(ctx); ok {
			// Ignore retransmission.
			return
		}
		cer := new(smparser.CER)
		failedAVP, err := cer.Parse(m)
		if err != nil {
			if failedAVP != nil {
				err = errorCEA(sm, c, m, cer, failedAVP)
				if err != nil {
					sm.Error(&diam.ErrorReport{
						Conn:    c,
						Message: m,
						Error:   err,
					})
				}
			}
			c.Close()
			return
		}
		err = successCEA(sm, c, m, cer)
		if err != nil {
			sm.Error(&diam.ErrorReport{
				Conn:    c,
				Message: m,
				Error:   err,
			})
			return
		}
		meta := smpeer.FromCER(cer)
		c.SetContext(smpeer.NewContext(ctx, meta))
		// Notify about peer passing the handshake.
		select {
		case sm.hsNotifyc <- c:
		default:
		}
	}
}

// errorCEA sends an error answer indicating that the CER failed due to
// an unsupported (acct/auth) application, and includes the AVP that
// caused the failure in the message.
func errorCEA(sm *StateMachine, c diam.Conn, m *diam.Message, cer *smparser.CER, failedAVP *diam.AVP) error {
	hostIP, _, err := net.SplitHostPort(c.LocalAddr().String())
	if err != nil {
		return fmt.Errorf("failed to parse own ip %q: %s", c.LocalAddr(), err)
	}
	var a *diam.Message
	if failedAVP == cer.InbandSecurityID {
		a = m.Answer(diam.NoCommonSecurity)
	} else {
		a = m.Answer(diam.NoCommonApplication)
	}
	a.Header.CommandFlags |= diam.ErrorFlag
	a.NewAVP(avp.OriginHost, avp.Mbit, 0, sm.cfg.OriginHost)
	a.NewAVP(avp.OriginRealm, avp.Mbit, 0, sm.cfg.OriginRealm)
	a.NewAVP(avp.HostIPAddress, avp.Mbit, 0, datatype.Address(net.ParseIP(hostIP)))
	a.NewAVP(avp.VendorID, avp.Mbit, 0, sm.cfg.VendorID)
	a.NewAVP(avp.ProductName, 0, 0, sm.cfg.ProductName)
	if cer.OriginStateID != nil {
		a.AddAVP(cer.OriginStateID)
	}
	a.NewAVP(avp.FailedAVP, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{failedAVP},
	})
	if sm.cfg.FirmwareRevision != 0 {
		a.NewAVP(avp.FirmwareRevision, avp.Mbit, 0, sm.cfg.FirmwareRevision)
	}
	_, err = a.WriteTo(c)
	return err
}

// successCEA sends a success answer indicating that the CER was successfully
// parsed and accepted by the server.
func successCEA(sm *StateMachine, c diam.Conn, m *diam.Message, cer *smparser.CER) error {
	hostIP, _, err := net.SplitHostPort(c.LocalAddr().String())
	if err != nil {
		return fmt.Errorf("failed to parse own ip %q: %s", c.LocalAddr(), err)
	}
	a := m.Answer(diam.Success)
	a.NewAVP(avp.OriginHost, avp.Mbit, 0, sm.cfg.OriginHost)
	a.NewAVP(avp.OriginRealm, avp.Mbit, 0, sm.cfg.OriginRealm)
	a.NewAVP(avp.HostIPAddress, avp.Mbit, 0, datatype.Address(net.ParseIP(hostIP)))
	a.NewAVP(avp.VendorID, avp.Mbit, 0, sm.cfg.VendorID)
	a.NewAVP(avp.ProductName, 0, 0, sm.cfg.ProductName)
	if cer.OriginStateID != nil {
		a.AddAVP(cer.OriginStateID)
	}
	if cer.AcctApplicationID != nil {
		for _, acct := range cer.AcctApplicationID {
			a.AddAVP(acct)
		}
	}
	if cer.AuthApplicationID != nil {
		for _, auth := range cer.AuthApplicationID {
			a.AddAVP(auth)
		}
	}
	if cer.VendorSpecificApplicationID != nil {
		for _, vs := range cer.VendorSpecificApplicationID {
			a.AddAVP(vs)
		}
	}
	if sm.cfg.FirmwareRevision != 0 {
		a.NewAVP(avp.FirmwareRevision, avp.Mbit, 0, sm.cfg.FirmwareRevision)
	}
	_, err = a.WriteTo(c)
	return err
}
