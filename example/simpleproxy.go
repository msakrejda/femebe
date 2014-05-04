package main

import (
	"fmt"
	"github.com/uhoh-itsmaciek/femebe"
	"github.com/uhoh-itsmaciek/femebe/core"
	"github.com/uhoh-itsmaciek/femebe/proto"
	"github.com/uhoh-itsmaciek/femebe/util"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

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

	target := os.Args[2]
	resolver := &fixedResolver{target}
	manager := femebe.NewSimpleSessionManager()
	p := &proxy{resolver, manager}

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		log.Print("accepting connection")

		go p.handleConnection(conn, target)
	}
}

type proxy struct {
	resolver femebe.Resolver
	manager  femebe.SessionManager
}

type fixedResolver struct {
	targetAddr string
}

func (pr *fixedResolver) Resolve(params map[string]string) femebe.Connector {
	return femebe.NewSimpleConnector(pr.targetAddr, params)
}

func (p *proxy) handleConnection(conn net.Conn, serverAddr string) {
	var err error
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("error in handling connection: %v", p)
			conn.Close()
		} else {
			// Log disconnections: right now, we still return EOF
			// errors for connections that are closed "cleanly"
			// (e.g., client disconnects or server crashes) rather
			// than due to any problems encountered during
			// protocol manipulation. We assume that any
			// connectivity errors are "clean" and we ignore them
			// for now.
			if err != nil && err != io.EOF {
				log.Print("Session exits with error: ", err)
			} else {
				log.Print("Session exits cleanly")
			}
		}
	}()

	feStream := core.NewFrontendStream(util.NewBufferedReadWriteCloser(conn))
	var m core.Message
	err = feStream.Next(&m)
	if err != nil {
		panic(fmt.Errorf("could not read client startup message: %v", err))
	}
	if proto.IsStartupMessage(&m) {
		startup, err := proto.ReadStartupMessage(&m)
		if err != nil {
			panic(fmt.Errorf("could not parse client startup message: %v", err))
		}
		connector := p.resolver.Resolve(startup.Params)
		beStream, err := connector.Startup()
		if err != nil {
			panic(fmt.Errorf("could not connect to backend: %v", err))
		}
		router := femebe.NewSimpleRouter(feStream, beStream)
		session := femebe.NewSimpleSession(router, connector)
		err = p.manager.RunSession(session)
	} else if proto.IsSSLRequest(&m) {
		log.Print("SSL not supported; try with PGSSLMODE=disable")
	} else if proto.IsCancelRequest(&m) {
		cancel, err := proto.ReadCancelRequest(&m)
		if err != nil {
			panic(fmt.Errorf("could not parse cancel message: %v", err))
		}
		err = p.manager.Cancel(cancel.BackendPid, cancel.SecretKey)
		if err != nil {
			panic(fmt.Errorf("could not process cancellation: %v", err))
		}
		err = conn.Close()
		if err != nil {
			fmt.Println(err)
		}
	} else {
		panic(fmt.Errorf("could not understand client"))
	}
}
