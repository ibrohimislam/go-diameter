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
)

// A Client is a diameter client that automatically performs a handshake
// with connected client after the connection is accepted.

type Server struct {
	Dict                        *dict.Parser  // Dictionary parser (uses dict.Default if unset)
	Handler                     *StateMachine // Message handler
	SupportedVendorID           []*diam.AVP   // Supported vendor ID
	AcctApplicationID           []*diam.AVP   // Acct applications
	AuthApplicationID           []*diam.AVP   // Auth applications
	VendorSpecificApplicationID []*diam.AVP   // Vendor specific applications
}
