package femebe

import (
	"bytes"
	"testing"
)

//Benchmark for testing how long the send operation on a MessageStream takes
func BenchmarkEchoSend(b *testing.B) {
	b.StopTimer()
	var ping Message
	var pong Message

	ping.InitFromBytes('i', []byte("ftest"))
	buf := NewPackBuffer(9208)
	ms := NewServerMessageStream("echo", buf)

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		for j := 0; j < 1000; j++ {
			ms.Send(&ping)
		}
		b.StopTimer()
		for j := 0; j < 1000; j++ {
			ms.Next(&pong)
		}
	}
}

//Benchmark for testing how long the send operation on a MessageStream takes
func BenchmarkEchoNext(b *testing.B) {
	b.StopTimer()
	var ping Message
	var pong Message

	ping.InitFromBytes('i', []byte("ftest"))
	underBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	buf := newClosableBuffer(underBuf)
	ms := NewServerMessageStream("echo", buf)

	for i := 0; i < b.N; i++ {
		underBuf.Reset()
		for j := 0; j < 1000; j++ {
			ms.Send(&ping)
		}
		b.StartTimer()
		for j := 0; j < 1000; j++ {
			ms.Next(&pong)
		}
		b.StopTimer()
	}

}

//Test function to make sure that everything is working before benching
func TestEcho(t *testing.T) {
	var ping Message
	ping.InitFromBytes('i', []byte("ftest"))
	var pong Message

	buf := NewPackBuffer(1024)

	ms := NewServerMessageStream("echo", buf)
	print("sending")
	ms.Send(&ping)
	t.Logf("%v", buf)

	print("recv")
	ms.Next(&pong)

	rest, _ := pong.Force()
	t.Logf("Type:%c, bytes:%s,", pong.MsgType(), rest)
}
