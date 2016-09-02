// Copyright 2013-2015 go-diameter authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package smparser

import (
	"strings"
	"testing"

	"github.com/ibrohimislam/go-diameter/diam"
	"github.com/ibrohimislam/go-diameter/diam/avp"
	"github.com/ibrohimislam/go-diameter/diam/datatype"
	"github.com/ibrohimislam/go-diameter/diam/dict"
)

// These tests use a custom dictionary loaded by sm_test.go.

func TestCER_MissingOriginHost(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	if err != ErrMissingOriginHost {
		t.Fatal("Unexpected error:", err)
	}
}

func TestCER_MissingOriginRealm(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	if err != ErrMissingOriginRealm {
		t.Fatal("Unexpected error:", err)
	}
}

func TestCER_MissingApplication(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	if err != ErrMissingApplication {
		t.Fatal("Unexpected error:", err)
	}
}

func TestCER_NoCommonApplication(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(2))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	appErr, ok := err.(*ErrNoCommonApplication)
	if !ok {
		t.Fatal("Unexpected error:", err.Error())
	}
	if appErr.ID != 2 {
		t.Fatalf("Unexpected app ID. Want 2, have %d", appErr.ID)
	}
	if !strings.Contains(appErr.Error(), "acct application 2") {
		t.Fatalf("Unexpected error message: %s", appErr)
	}
}

func TestCER_NoCommonSecurity(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.InbandSecurityID, avp.Mbit, 0, datatype.Unsigned32(1))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	if err != ErrNoCommonSecurity {
		t.Fatal("Unexpected error:", err)
	}
}

func TestCER_AcctAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(1001))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err != nil {
		t.Fatal(err)
	}
	if app := cer.Applications(); len(app) != 1 {
		if app[0] != 1001 {
			t.Fatalf("Unexpected app ID. Want 1001, have %d", app[0])
		}
	}
}

func TestCER_FailedAcctAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(1000))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	appErr, ok := err.(*ErrNoCommonApplication)
	if !ok {
		t.Fatal(err)
	}
	if appErr.ID != 1000 {
		t.Fatalf("Unexpected app ID. Want 1000, have %d", appErr.ID)
	}
}

func TestCER_AcctNotAuthAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(1001))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	appErr, ok := err.(*ErrNoCommonApplication)
	if !ok {
		t.Fatal(err)
	}
	if appErr.ID != 1001 {
		t.Fatalf("Unexpected app ID. Want 1001, have %d", appErr.ID)
	}
}

func TestCER_AuthAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(1002))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err != nil {
		t.Fatal(err)
	}
	if app := cer.Applications(); len(app) != 1 {
		if app[0] != 1002 {
			t.Fatalf("Unexpected app ID. Want 1002, have %d", app[0])
		}
	}
}

func TestCER_FailedAuthAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(1000))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	appErr, ok := err.(*ErrNoCommonApplication)
	if !ok {
		t.Fatal(err)
	}
	if appErr.ID != 1000 {
		t.Fatalf("Unexpected app ID. Want 1000, have %d", appErr.ID)
	}
}

func TestCER_AuthNotAcctAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(1002))
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	appErr, ok := err.(*ErrNoCommonApplication)
	if !ok {
		t.Fatal(err)
	}
	if appErr.ID != 1002 {
		t.Fatalf("Unexpected app ID. Want 1002, have %d", appErr.ID)
	}
}

func TestCER_VSAcctAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(1001)),
		},
	})
	cer := new(CER)
	_, err := cer.Parse(m)
	if err != nil {
		t.Fatal(err)
	}
	if app := cer.Applications(); len(app) != 1 {
		if app[0] != 1001 {
			t.Fatalf("Unexpected app ID. Want 1001, have %d", app[0])
		}
	}
}

func TestCER_FailedVSAcctAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(1000)),
		},
	})
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	appErr, ok := err.(*ErrNoCommonApplication)
	if !ok {
		t.Fatal(err)
	}
	if appErr.ID != 1000 {
		t.Fatalf("Unexpected app ID. Want 1000, have %d", appErr.ID)
	}
}

func TestCER_VSAuthAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(1002)),
		},
	})
	cer := new(CER)
	_, err := cer.Parse(m)
	if err != nil {
		t.Fatal(err)
	}
	if app := cer.Applications(); len(app) != 1 {
		if app[0] != 1002 {
			t.Fatalf("Unexpected app ID. Want 1002, have %d", app[0])
		}
	}
}

func TestCER_FailedVSAuthAppID(t *testing.T) {
	m := diam.NewRequest(diam.CapabilitiesExchange, 0, dict.Default)
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("foobar"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("test"))
	m.NewAVP(avp.OriginStateID, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(1000)),
		},
	})
	cer := new(CER)
	_, err := cer.Parse(m)
	if err == nil {
		t.Fatal("Broken CER was parsed with no errors")
	}
	appErr, ok := err.(*ErrNoCommonApplication)
	if !ok {
		t.Fatal(err)
	}
	if appErr.ID != 1000 {
		t.Fatalf("Unexpected app ID. Want 1000, have %d", appErr.ID)
	}
}
