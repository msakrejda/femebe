package femebe

import (
	"testing"
)

//Benchmark for testing how long the send operation on a MessageStream takes
func BenchmarkEchoSend(b *testing.B) {
	b.StopTimer()
	var ping Message
	var pong Message

	ping.InitFromBytes('i', make([]byte, 50))
	buf := NewPackBuffer(2048)
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

	ping.InitFromBytes('i', make([]byte, 50))
	buf := NewPackBuffer(204800)
	ms := NewServerMessageStream("echo", buf)

	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			ms.Send(&ping)
		}
		b.StartTimer()
		for j := 0; j < 10000; j++ {

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
	t.Logf("Type:%v, bytes:%v,", pong.MsgType(), rest)
}
