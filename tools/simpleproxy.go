package main

import (
	"bufio"
	"femebe"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"crypto/tls"
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

func NewSimpleProxySession(
	errch chan error,
	client *femebe.MessageStream,
	server *femebe.MessageStream) *session {

	ingress := func() {
		var m femebe.Message

		for {
			err := client.Next(&m)
			if err != nil {
				errch <- err
				return
			}

			err = server.Send(&m)
			if err != nil {
				errch <- err
				return
			}

			if !client.HasNext() {
				server.Flush()
			}
		}
	}

	egress := func() {
		var m femebe.Message

		for {
			err := server.Next(&m)
			if err != nil {
				errch <- err
				return
			}

			err = client.Send(&m)
			if err != nil {
				errch <- err
				return
			}

			if !server.HasNext() {
				client.Flush()
			}
		}
	}

	return &session{ingress: ingress, egress: egress}
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
func handleConnection(clientConn net.Conn, serverAddr string) {
	var err error

	// Log disconnections
	defer func() {
		if err != nil && err != io.EOF {
			fmt.Printf("Session exits with error: %v\n", err)
		} else {
			fmt.Printf("Session exits cleanly\n")
		}
	}()

	defer clientConn.Close()

	c := femebe.NewClientMessageStream(
		"Client", newBufWriteCon(clientConn))

	unencryptServerConn, err := autoDial(serverAddr)
	if err != nil {
		fmt.Printf("Could not connect to server: %v\n", err)
	}

	tlsConf := tls.Config{}
	tlsConf.InsecureSkipVerify = true

	sConn, err := femebe.NegotiateTLS(
		unencryptServerConn, "prefer", &tlsConf)
	if err != nil {
		fmt.Printf("Could not negotiate TLS: %v\n", err)
	}

	s, err := femebe.NewServerMessageStream(
		"Server", newBufWriteCon(sConn))
	if err != nil {
		fmt.Printf("Could not initialize connection to server: %v\n", err)
	}

	done := make(chan error)
	NewSimpleProxySession(done, c, s).start()
	err = <-done
}

// Startup and main client acceptance loop
func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: simpleproxy LISTENADDR SERVERADDR\n")
		os.Exit(1)
	}

	ln, err := autoListen(os.Args[1])
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
