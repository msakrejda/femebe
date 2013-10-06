package main

import (
	"bufio"
	"crypto/tls"
	"github.com/deafbybeheading/femebe"
	"github.com/deafbybeheading/femebe/util"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
)

type session struct {
	ingress func()
	egress  func()
}

func (s *session) start() {
	go s.ingress()
	go s.egress()
}

type ProxyPair struct {
	*femebe.MessageStream
	net.Conn
}

func NewSimpleProxySession(errch chan error,
	fe *ProxyPair, be *ProxyPair) *session {
	mover := func(from, to *ProxyPair) func() {
		return func() {
			var err error

			defer func() {
				from.Close()
				to.Close()
				errch <- err
			}()

			var m femebe.Message

			for {
				err = from.Next(&m)
				if err != nil {
					return
				}

				err = to.Send(&m)
				if err != nil {
					return
				}

				if !from.HasNext() {
					err = to.Flush()
					if err != nil {
						return
					}
				}
			}
		}
	}

	return &session{
		ingress: mover(fe, be),
		egress:  mover(be, fe),
	}
}

type bufWriteCon struct {
	io.ReadCloser
	femebe.Flusher
	io.Writer
}

func newBufWriteCon(c net.Conn) *bufWriteCon {
	bw := bufio.NewWriter(c)
	return &bufWriteCon{c, bw, bw}
}


// so...
/*
func handle(conn net.Conn) {
	feStream := NewFEStream(conn)
	var m Message
	err := feStream.Next(m)
	if err != nil {
		// ...
	}
	if message.IsStartup(m) {
		startup, err := message.ReadStartup(&m)
		if err != nil {
			// ... 
		}
		connector, err := resolver.Resolve(m.Version, m.Options)
		if err != nil {
			// ...
		}
		beStream, err := connector.Startup()
		if err != nil {
			// ...
		}

		router := femebe.NewSimpleRouter(feStream, beStream)
		session := femebe.NewSimpleSession(router, connector)

		go manager.RunSession(session)

	} else if message.IsCancel(m) {
		cancel, err := message.ReadCancel(&m)
		if err != nil {
			// ... 
		}
		go manager.Cancel(cancel.BackendPid, cancel.SecretKey)
	} else {
		// unknown message type: we can't do anything with this
		_ = conn.Close()
	}
}
*/

// Generic connection handler
//
// This redelegates to more specific proxy handlers that contain the
// main proxy loop logic.
func handleConnection(feConn net.Conn, serverAddr string) {
	var err error

	// Log disconnections
	defer func() {
		if err != nil && err != io.EOF {
			fmt.Printf("Session exits with error: %v\n", err)
		} else {
			fmt.Printf("Session exits cleanly\n")
		}
	}()

	defer feConn.Close()

	c := femebe.NewFrontendStream(newBufWriteCon(feConn))

	unencryptedBeConn, err := util.AutoDial(serverAddr)
	if err != nil {
		fmt.Printf("Could not connect to server: %v\n", err)
	}

	conf := &util.SSLConfig{Mode: util.SSLPrefer, Config: tls.Config{InsecureSkipVerify: true}}
	beConn, err := util.NegotiateTLS(unencryptedBeConn, conf)
	if err != nil {
		fmt.Printf("Could not negotiate TLS: %v\n", err)
	}

	s := femebe.NewBackendStream(newBufWriteCon(beConn))
	if err != nil {
		fmt.Printf("Could not initialize connection to server: %v\n", err)
	}

	done := make(chan error)
	NewSimpleProxySession(done, &ProxyPair{c, feConn},
		&ProxyPair{s, beConn}).start()

	// Both sides must exit to finish
	_ = <-done
	_ = <-done
}

// Startup and main client acceptance loop
func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: simpleproxy LISTENADDR SERVERADDR\n")
		os.Exit(1)
	}

	ln, err := util.AutoListen(os.Args[1])
	if err != nil {
		fmt.Printf("Could not listen on address: %v\n", err)
		os.Exit(1)
	}

	// Signal handling; this is pretty ghetto now, but at least we
	// can exit cleanly on an interrupt. N.B.: this currently does
	// not correctly capture SIGTERM on Linux (and possibly
	// elsewhere)--it just kills the process directly without
	// involving the signal handler.
	sigch := make(chan os.Signal)
	signal.Notify(sigch, os.Interrupt, os.Kill)
	watchSigs := func() {
		for sig := range sigch {
			fmt.Printf("Got signal %v", sig)
			if sig == os.Kill {
				os.Exit(2)
			} else if sig == os.Interrupt {
				os.Exit(0)
			}
		}
	}
	go watchSigs()

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		go handleConnection(conn, os.Args[2])
	}

	fmt.Println("simpleproxy quits successfully")
	return
}
