// SPDX-License-Identifier: MIT

package roomsrv

import (
	"fmt"
	"net"

	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
)

// opens the shs listener for TCP connections
func (s *Server) initNetwork() error {
	// muxrpc handler creation and authoratization decider
	mkHandler := func(conn net.Conn) (muxrpc.Handler, error) {
		s.closedMu.Lock()
		defer s.closedMu.Unlock()

		remote, err := network.GetFeedRefFromAddr(conn.RemoteAddr())
		if err != nil {
			return nil, fmt.Errorf("sbot: expected an address containing an shs-bs addr: %w", err)
		}

		if s.keyPair.Feed.Equal(remote) {
			return &s.master, nil
		}

		pm, err := s.Config.GetPrivacyMode(nil)
		if err != nil {
			return nil, fmt.Errorf("running with unknown privacy mode")
		}

		// if privacy mode is restricted, deny connections from non-members
		if pm == roomdb.ModeRestricted {
			if _, err := s.authorizer.GetByFeed(s.rootCtx, *remote); err != nil {
				return nil, fmt.Errorf("access restricted to members")
			}
		}

		// for community + open modes, allow all connections
		return &s.public, nil
	}

	// tcp+shs
	opts := network.Options{
		Logger:              s.logger,
		Dialer:              s.dialer,
		ListenAddr:          s.listenAddr,
		KeyPair:             s.keyPair,
		AppKey:              s.appKey[:],
		MakeHandler:         mkHandler,
		ConnTracker:         s.networkConnTracker,
		BefreCryptoWrappers: s.preSecureWrappers,
		AfterSecureWrappers: s.postSecureWrappers,

		WebsocketAddr: s.wsAddr,
	}

	var err error
	s.Network, err = network.New(opts)
	if err != nil {
		return fmt.Errorf("failed to create network node: %w", err)
	}

	return nil
}
