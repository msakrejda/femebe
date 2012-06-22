package femebe

import (
	"fmt"
	"net"
	"testing"
)

type FakeReader struct{}
type FakeWriter struct{}

func (*FakeReader) Read(p []byte) (n int, err error) {
	for i, _ := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func (*FakeWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// func TestFEBEConn(t *testing.T) {
// 	conn := NewMessageStream(&FakeReader{}, &FakeWriter{})
// 	conn.Next()
// }

func TestListen(t *testing.T) {
	makeServer(t)
}

func handleConnection2(conn net.Conn) {
	fmt.Println("processing request")

	c := NewMessageStream("Conn", conn, conn)
	for {
		message, _ := c.Next()
		fmt.Printf("got message %c\n", message.msgType)
		msg := NewAuthenticationOk()
		sendMsg(c, msg, nil)
		msg, err := NewReadyForQuery(RFQ_IDLE)
		sendMsg(c, msg, err)
		// read in a query
		message, _ = c.Next()
		msg = NewRowDescription([]FieldDescription{
			*NewField("col1", STRING),
			*NewField("col2", STRING),
		})
		sendMsg(c, msg, err)
		msg = NewDataRow([]interface{}{"hello", "world"})
		sendMsg(c, msg, err)
		msg = NewCommandComplete("SELECT 1")
		sendMsg(c, msg, err)
		msg, err = NewReadyForQuery(RFQ_IDLE)
		sendMsg(c, msg, err)
	}
}

func server2client(client MessageStream, server MessageStream, errch chan error) {
	for {
		msg, err := server.Next()
		if err != nil {
			fmt.Printf("Error %v\n", err)
			errch <- err
			return
		}
		err = client.Send(msg)
		if err != nil {
			fmt.Printf("Error %v\n", err)
			errch <- err
			return
		}
	}
}

func client2server(client MessageStream, server MessageStream, errch chan error) {
	for {
		msg, err := client.Next()
		if err != nil {
			fmt.Printf("Error %v\n", err)
			errch <- err
			return
		}
		err = server.Send(msg)
		if err != nil {
			fmt.Printf("Error %v\n", err)
			errch <- err
			return
		}
	}
}

func handleConnection3(conn net.Conn) {
	fmt.Println("processing request")

	c := NewMessageStream("Client", conn, conn)

	serverConn, err := net.Dial("tcp", "localhost:5434")
	if err != nil {
		fmt.Println("oh snap")
	}
	s := NewMessageStream("Server", serverConn, serverConn)

	end := make(chan error)
	go server2client(c, s, end)
	go client2server(c, s, end)
	_ = <- end
	_ = <- end
}



func server2client2(client MessageStream, server MessageStream, errch chan error) {
	for {
		msg, err := server.Next()
		if err != nil {
			fmt.Printf("Error %v\n", err)
			errch <- err
			return
		}
		err = client.Send(msg)
		if err != nil {
			fmt.Printf("Error %v\n", err)
			errch <- err
			return
		}
	}
}

func client2server2(client MessageStream, server MessageStream, errch chan error) {
	for {
		msg, err := client.Next()
		if err != nil {
			fmt.Printf("Error %v\n", err)
			errch <- err
			return
		}
		if msg.msgType == '\000' {
			startupMsg := ReadStartupMessage(msg)
			fmt.Println("Got startup message:")
			for key, value := range startupMsg.params {
				fmt.Printf("\t%v: %v\n", key, value)
			}
		}
		err = server.Send(msg)
		if err != nil {
			fmt.Printf("Error %v\n", err)
			errch <- err
			return
		}
	}
}


func handleConnection(conn net.Conn) {
	fmt.Println("processing request")

	c := NewMessageStream("Client", conn, conn)

	// for some configured servers, list their databases; when a
	// startup packet comes in

	// select datname from pg_database

	msg, _ := c.Next()
	// Sanity-check the startup message by reading it
	ReadStartupMessage(msg)
	
	sConn1, messages1, _ := try("localhost:5435", msg)
	sConn2, messages2, _ := try("localhost:5434", msg)

	var s MessageStream
	var m []*Message

	if sConn1 != nil {
		s = sConn1
		m = messages1
	} else if sConn2 != nil {
		s = sConn2
		m = messages2
	} else {
		panic("Oh snap!")
	}

	for _, msg := range m {
		c.Send(msg)
	}

	end := make(chan error)
	go server2client2(c, s, end)
	go client2server2(c, s, end)
	_ = <- end
	_ = <- end
}

func try(connstr string, authMsg *Message) (stream MessageStream, messages []*Message, err error) {
	conn, err := net.Dial("tcp", connstr)
	s := NewMessageStream("Server", conn, conn)
	if err != nil {
		goto onError
	}

	s.Send(authMsg)

	messages = make([]*Message, 0, 5)
	for {
		nextMsg, err := s.Next()
		if err != nil {
			goto onError
		}

		messages = append(messages, nextMsg)
		if IsReadyForQuery(nextMsg) {
			return s, messages, nil
		}
	}

	panic("Oh snap! We ran out of infinity!")

onError:
	return nil, nil, err
}

func sendMsg(stream MessageStream, msg *Message, err error) {
	if err != nil {
		fmt.Printf("Could not build message: %v", err)
	}
	e := stream.Send(msg)
	if e != nil {
		fmt.Printf("Unable to send message %v: %v", msg, e)
	}
	fmt.Println("Sent")
}

func makeServer(t *testing.T) {
	ln, err := net.Listen("tcp", ":5432")
	if err != nil {
		t.Fatal("can't listen")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			t.Fatalf("Error accepting connection %v", err)
		}
		go handleConnection(conn)
	}
}
