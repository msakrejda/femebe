package main

import (
	"bufio"
	"crypto/tls"
	"femebe"
	"femebe/pgproto"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
)

// Automatically chooses between unix sockets and tcp sockets for
// listening
func autoListen(place string) (net.Listener, error) {
	if strings.Contains(place, "/") {
		return net.Listen("unix", place)
	}

	return net.Listen("tcp", place)
}

// Automatically chooses between unix sockets and tcp sockets for
// dialing.
func autoDial(place string) (net.Conn, error) {
	if strings.Contains(place, "/") {
		return net.Dial("unix", place)
	}

	return net.Dial("tcp", place)
}

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
	client *ProxyPair, server *ProxyPair) *session {
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
		ingress: mover(client, server),
		egress:  mover(server, client),
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

// Generic connection handler
//
// This redelegates to more specific proxy handlers that contain the
// main proxy loop logic.
func handleConnection(cConn net.Conn, rt *routingTable) {
	var err error

	// Log disconnections
	defer func() {
		if err != nil && err != io.EOF {
			log.Printf("Session exits with error: %v\n", err)
		} else {
			log.Printf("Session exits cleanly\n")
		}
	}()

	defer cConn.Close()

	c := femebe.NewClientMessageStream(
		"Client", newBufWriteCon(cConn))

	// Must interpret Startup and Cancel requests.
	//
	// SSL Negotiation requests not handled for now.
	var firstPacket femebe.Message
	c.Next(&firstPacket)

	// Handle Startup packets
	var sup *pgproto.Startup
	if sup, err = pgproto.ReadStartupMessage(&firstPacket); err != nil {
		log.Print(err)
		return
	}

	var ent *routingEntry
	if ent = rt.rewrite(sup); ent == nil {
		log.Print("Could not route startup packet")
		return
	}

	unencryptServerConn, err := autoDial(ent.addr)
	if err != nil {
		log.Printf("Could not connect to server: %v\n", err)
		return
	}

	tlsConf := tls.Config{}
	tlsConf.InsecureSkipVerify = true

	sConn, err := femebe.NegotiateTLS(
		unencryptServerConn, "prefer", &tlsConf)
	if err != nil {
		log.Printf("Could not negotiate TLS: %v\n", err)
		return
	}

	s := femebe.NewServerMessageStream("Server", newBufWriteCon(sConn))
	if err != nil {
		log.Printf("Could not initialize connection to server: %v\n", err)
		return
	}

	var rewrittenStatupMessage femebe.Message
	sup.FillMessage(&rewrittenStatupMessage)
	err = s.Send(&rewrittenStatupMessage)
	if err != nil {
		return
	}

	err = s.Flush()
	if err != nil {
		return
	}

	done := make(chan error)
	NewSimpleProxySession(done,
		&ProxyPair{c, cConn},
		&ProxyPair{s, sConn}).start()

	// Both sides must exit to finish
	_ = <-done
	_ = <-done
}

func parseRoutingEntry(tupleRaw string) (*routingEntry, error) {
	parts := strings.Split(tupleRaw, ",")

	const TUPSZ = 3
	if partL := len(parts); partL != TUPSZ {
		return nil, fmt.Errorf(
			"Bad routing tuple: expect len %v, "+
				"got len %v for tuple text %v",
			TUPSZ, partL, tupleRaw)
	}

	return &routingEntry{parts[0], parts[1], parts[2]}, nil
}

// Signal handling: this is pretty ghetto now, but at least we can
// exit cleanly on an interrupt. N.B.: this currently does not
// correctly capture SIGTERM on Linux (and possibly elsewhere)--it
// just kills the process directly without involving the signal
// handler.
func installSignalHandlers() {
	sigch := make(chan os.Signal)
	signal.Notify(sigch, os.Interrupt, os.Kill)
	go func() {
		for sig := range sigch {
			log.Printf("Got signal %v", sig)
			if sig == os.Kill {
				os.Exit(2)
			} else if sig == os.Interrupt {
				os.Exit(0)
			}
		}
	}()
}

// Startup and main client acceptance loop
func main() {
	installSignalHandlers()

	if len(os.Args) < 3 {
		log.Printf(
			"Usage: dog LISTENADDR " +
				"(DBNAMEIN,ADDR,DBNAMEOUT)+")
		os.Exit(1)
	}

	ln, err := autoListen(os.Args[1])
	if err != nil {
		log.Printf("Could not listen on address: %v", err)
		os.Exit(1)
	}

	rt := newRoutingTable()
	for _, rawTup := range os.Args[2:] {
		re, err := parseRoutingEntry(rawTup)
		if err != nil {
			log.Fatal(err)
		} else {
			rt.post(re)
		}
	}

	for {
		conn, err := ln.Accept()

		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		go handleConnection(conn, rt)
	}

	log.Println("simpleproxy quits successfully")
	return
}
