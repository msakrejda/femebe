package main

import (
	"femebe"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
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
		var msg femebe.Message

		for {
			err := client.Next(&msg)
			if err != nil {
				errch <- err
				return
			}

			err = server.Send(&msg)
			if err != nil {
				errch <- err
				return
			}
		}
	}

	egress := func() {
		var msg femebe.Message

		for {
			err := server.Next(&msg)
			if err != nil {
				errch <- err
				return
			}

			err = client.Send(&msg)
			if err != nil {
				errch <- err
				return
			}
		}
	}

	return &session{ingress: ingress, egress: egress}
}

// Virtual hosting connection handler
func proxyHandler(clientConn net.Conn, rt *RoutingTable) {
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

	c := femebe.NewClientMessageStream("Client", clientConn)

	// Handle the very first message -- the startup packet --
	// specially to do switching.
	var firstMessage femebe.Message
	if err = c.Next(&firstMessage); err != nil {
		return
	}

	startupMsg, err := firstMessage.ReadStartupMessage()
	if err != nil {
		return
	}

	dbname := startupMsg.Params["database"]
	serverAddr := rt.Route(dbname)

	// No route found, quickly exit
	if serverAddr == "" {
		fmt.Printf("No route found for database \"%v\"\n", dbname)
		return
	}

	// Route was found, so now start a trivial proxy forwarding
	// traffic.
	serverConn, err := autoDial(serverAddr)
	if err != nil {
		fmt.Printf("Could not connect to server: %v\n", err)
	}

	s, err := femebe.NewServerMessageStream("Server", serverConn)
	if err = s.Send(&firstMessage); err != nil {
		fmt.Printf("Could not relay startup packet: %v\n", err)
	}

	done := make(chan error)
	NewSimpleProxySession(done, c, s).start()

	err = <-done
}

func NewAdminSession(
	errch chan error,
	client *femebe.MessageStream,
	rt *RoutingTable) *session {
	commandCh := make(chan string)

	ingress := func() {
		var msg femebe.Message

		for {
			err := client.Next(&msg)

			if err != nil {
				errch <- err
				return
			}

			if femebe.IsQuery(&msg) {
				q, err := femebe.ReadQuery(&msg)
				if err != nil {
					return
				}

				commandCh <- q.Query
			}
		}
	}

	dumpRoutingTable := func() {
		data := make([][]interface{}, 0, 1000)

		fmt.Printf("rt.table: %#q\n", rt.table)
		for key, value := range rt.table {
			data = append(data, []interface{}{key, value})
		}

		SendDataSet(client, []string{"key", "value"}, data)
	}

	egress := func() {
		// Obligatory steps to start up.  The admin connection
		// is currently on a trust-only basis (no
		// authentication), intended to be used soley via unix
		// socket, or not at all.
		var msg femebe.Message

		msg.InitAuthenticationOk()
		client.Send(&msg)

		msg.InitReadyForQuery(femebe.RFQ_IDLE)
		client.Send(&msg)

		for {
			command := <-commandCh

			if command == "dump routing table;" {
				dumpRoutingTable()
				msg.InitReadyForQuery(femebe.RFQ_IDLE)
				client.Send(&msg)
			} else {
				fmt.Printf("Ignoring unknown command %v\n", command)
			}

		}
	}

	return &session{ingress: ingress, egress: egress}
}

func adminHandler(clientConn net.Conn, rt *RoutingTable) {
	errch := make(chan error)
	ms := femebe.NewClientMessageStream("AdminClient", clientConn)
	NewAdminSession(errch, ms, rt).start()
	_ = <-errch
}

func SendDataSet(stream *femebe.MessageStream, colnames []string,
	rows [][]interface{}) {
	rowLen := len(colnames)
	fieldDescs := make([]femebe.FieldDescription, len(colnames))
	for i, name := range colnames {
		fieldDescs[i] = *femebe.NewField(name, femebe.STRING)
	}

	var msg femebe.Message
	msg.InitRowDescription(fieldDescs)
	stream.Send(&msg)
	fmt.Printf("%#q\n", rows)
	for _, row := range rows {
		fmt.Printf("%#q", row)
		if len(row) != rowLen {
			panic(fmt.Errorf("Oh snap: len(row) is %v, "+
				"but rowLen is %v\n", len(row), rowLen))
		}

		msg.InitDataRow(row)
		stream.Send(&msg)
	}

	msg.InitCommandComplete(fmt.Sprintf("SELECT %v", rowLen))
	stream.Send(&msg)
}

type Acceptor func(ln net.Conn)

func AcceptorLoop(ln net.Listener, a Acceptor, done chan bool) {
	defer func() { done <- true }()

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		go a(conn)
	}
}

type RoutingTable struct {
	table map[string]string
	l     sync.Mutex
}

func NewRoutingTable() (rt *RoutingTable) {
	return &RoutingTable{table: make(map[string]string)}
}

func (r *RoutingTable) SetRoute(dbname, addr string) {
	// Only necessary because hash tables are allowed to race and
	// subsequently uglifully crash in non-sandboxed Go programs.
	r.l.Lock()
	defer r.l.Unlock()

	r.table[dbname] = addr
}

func (r *RoutingTable) Route(dbname string) (addr string) {
	return r.table[dbname]
}

// Startup and main client acceptance loop
func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: toydog LISTENADDR\n")
		os.Exit(1)
	}

	listen := func(addr string) net.Listener {
		ln, err := autoListen(addr)
		if err != nil {
			fmt.Printf("Could not listen on address: %v\n", err)
			os.Exit(1)
		}
		return ln
	}
	proxyLn := listen(os.Args[1])
	if proxyLn != nil {
		defer proxyLn.Close()
	}

	adminLn := listen("/tmp/.s.PGSQL.45432")
	if adminLn != nil {
		defer adminLn.Close()
	}

	rt := NewRoutingTable()
	// The routing table is empty, which makes this program
	// useless.  One can experiment by adding things like:
	//
	rt.SetRoute("fdr", "/var/run/postgresql/.s.PGSQL.5432")

	done := make(chan bool)
	go AcceptorLoop(proxyLn,
		func(conn net.Conn) {
			proxyHandler(conn, rt)
		},
		done)

	go AcceptorLoop(adminLn,
		func(conn net.Conn) {
			adminHandler(conn, rt)
		},
		done)

	_ = <-done
	_ = <-done
	fmt.Println("toydog quits successfully")
	return
}
