package main

import (
	"strings"
	"bufio"
	"crypto/tls"
	"femebe"
	"femebe/pgproto"
	"log"
	"io"
	"net"
	"os"
	"os/signal"
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
	client     *femebe.MessageStream
	clientConn net.Conn
	server     *femebe.MessageStream
	serverConn net.Conn
}

func NewSimpleProxySession(errch chan error, pair *ProxyPair) *session {
	mover := func(from, to *femebe.MessageStream) func() {
		return func() {
			var err error

			defer func() {
				pair.serverConn.Close()
				pair.clientConn.Close()
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
		ingress: mover(pair.client, pair.server),
		egress:  mover(pair.server, pair.client),
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

// Dncapsulates error reporting when reading a startup packet.
func routeStartupAndPrintErrors() *routingEntry {
	s, err := pgproto.ReadStartupMessage(&firstPacket)
	if err != nil {
		log.Print(err)
		return nil
	}

	if dbnameIn := s["database"];  dbnameIn == "" {
		log.Print("no database name in startup packet")
	} else {
		if ent := rt.match(dbnameIn); ent == nil {
			log.Print("database name not found in " + 
				"routing table")
		} else {
			// Route found
			return ent
		}
	}

	return nil
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

	// Sanity check so .Force() is not destructive.
	if firstPacket.Size() > 10004 {
		log.Printf("refuse to handle large first-packet")
		return
	}

	if _, err := firstPacket.Force(); err != nil {
		log.Printf("could not read complete first-packet")
	}


	if ok, _ := pgproto.IsStartup(&firstPacket); ok {
		readStartupAndPrintErrors(&firstPacket)
	}
	

	unencryptServerConn, err := autoDial(serverAddr)
	if err != nil {
		log.Printf("Could not connect to server: %v\n", err)
	}

	tlsConf := tls.Config{}
	tlsConf.InsecureSkipVerify = true

	sConn, err := femebe.NegotiateTLS(
		unencryptServerConn, "require", &tlsConf)
	if err != nil {
		log.Printf("Could not negotiate TLS: %v\n", err)
	}

	s := femebe.NewServerMessageStream("Server", newBufWriteCon(sConn))
	if err != nil {
		log.Printf("Could not initialize connection to server: %v\n", err)
	}

	done := make(chan error)
	NewSimpleProxySession(done, &ProxyPair{
		client:     c,
		clientConn: cConn,
		server:     s,
		serverConn: sConn,
	}).start()

	// Both sides must exit to finish
	_ = <-done
	_ = <-done
}

func parseRoutingEntry(tupleRaw string) (*routingEntry, error) {
	parts := strings.Split(tupleRaw, ",")

	const TUPSZ = 3
	if partL := len(parts); partL != TUPSZ {
		return nil, fmt.Errorf(
			"Bad routing tuple: expect len %v, " +
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

	if len(os.Args) > 3 {
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
	for rawTup := range os.Args[2:] {
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
