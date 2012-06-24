package main

import (
	"femebe"
	"fmt"
	"io"
	"net"
	"os"
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

type handlerFunc func(
	client femebe.MessageStream,
	server femebe.MessageStream,
	errch chan error)

type proxyBehavior struct {
	toFrontend handlerFunc
	toServer   handlerFunc
}

func (pbh *proxyBehavior) start(
	client femebe.MessageStream,
	server femebe.MessageStream) (errch chan error) {

	errch = make(chan error)

	go pbh.toFrontend(client, server, errch)
	go pbh.toServer(client, server, errch)
	return errch
}

var simpleProxy = proxyBehavior{
	toFrontend: func(client femebe.MessageStream,
		server femebe.MessageStream, errch chan error) {
		for {
			msg, err := server.Next()
			if err != nil {
				errch <- err
				return
			}

			err = client.Send(msg)
			if err != nil {
				errch <- err
				return
			}
		}
	},
	toServer: func(client femebe.MessageStream,
		server femebe.MessageStream, errch chan error) {
		for {
			msg, err := client.Next()
			if err != nil {
				errch <- err
				return
			}

			err = server.Send(msg)
			if err != nil {
				errch <- err
				return
			}
		}
	},
}

// Generic connection handler
//
// This redelegates to more specific proxy handlers that contain the
// main proxy loop logic.
func handleConnection(proxy proxyBehavior,
	clientConn net.Conn, serverAddr string) {
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

	c := femebe.NewMessageStream("Client", clientConn, clientConn)

	serverConn, err := autoDial(serverAddr)
	if err != nil {
		fmt.Printf("Could not connect to server: %v\n", err)
	}

	b := femebe.NewMessageStream("Server", serverConn, serverConn)

	done := proxy.start(c, b)
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

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		go handleConnection(simpleProxy, conn, os.Args[2])
	}

	fmt.Println("simpleproxy quits successfully")
	return
}
