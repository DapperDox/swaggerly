/*
Copyright (C) 2016-2017 dapperdox.com 

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/
package network

import (
	"crypto/tls"
	"errors"
	"dapperdox/config"
	"dapperdox/logger"
	"net"
)

func GetListener(tlsEnabled *bool) (net.Listener, error) {

	cfg, _ := config.Get() // Don't worry about error. If there was something wrong with the config, we'd know by now.

	useTLS := 0
	if len(cfg.TLSCertificate) > 0 {
		useTLS++
	}
	if len(cfg.TLSKey) > 0 {
		useTLS++
	}

	// If no cert & key, then we're to run in plain-text mode
	if useTLS == 0 {
		logger.Infof(nil, "listening on %s for unsecured connections", cfg.BindAddr)
		return net.Listen("tcp", cfg.BindAddr)
	}

	if useTLS == 1 {
		return nil, errors.New("You must provide both a certificate and a key to enable TLS")
	}

	// Okay, we're building a TLS listener
	crt, err := tls.LoadX509KeyPair(cfg.TLSCertificate, cfg.TLSKey)
	if err != nil {
		return nil, err
	}

	// Be really secure!
	tlscfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		Certificates: []tls.Certificate{crt},
	}

	logger.Infof(nil, "listening on %s for SECURED connections", cfg.BindAddr)
	*tlsEnabled = true
	return tls.Listen("tcp", cfg.BindAddr, tlscfg)
}
