package dispatch

import (
	"errors"
	"github.com/deafbybeheading/femebe"
	"github.com/deafbybeheading/femebe/message"
	"github.com/deafbybeheading/femebe/util"
	"net"
	"sync"
)

type Scaffolding struct {
	HandleConnection(net.Conn)
}

type SessionManager interface {
	RunSession(session Session) error
	Cancel(backendPid, secretKey int32) error
}

type BackendKeyHolder interface {
	// Identifies this session, as per the FEBE protocol.
	// If this information is not yet available when this
	// method is called, or the router does not support
	// cancellation, it should return (-1, -1).
	BackendKeyData() (int32, int32)
}

// A Router moves protocol messages from the frontend to the backend
// (and vice versa). It is also responsible 
type Router interface {
	BackendKeyHolder
	// Route the next message from the frontend
	RouteFrontend() error
	// Route the next message from the bcakend
	RouteBackend() error
}

type Resolver interface {
	// Resolve a given startup message into a Connector that can be
	// used to connect to or send cancellations to the given backend
	Resolve(params map[string]string) Connector
}

// Can send (or delegate) a Postgres CancelRequest message. Should
// return an error if it knows that the request will not succeed. Note
// that due to the nature of the cancellation mechanism, there is no
// guarantee of success, so the absence of an error does not
// necessarily mean success.
type Canceller interface {
	// Open a stream to a backend and send a cancellation
	// request with the given data, the close the stream.
	Cancel(backendPid, secretKey int32) error
}

type Connector interface {
	Canceller
	// Open a stream to a backend, go through TLS negotiation (if
	// desired), and send a StartupMessage on that stream before
	// returning it. Return an error if a stream cannot be
	// established or if sending the startup packet returns an
	// error
	Startup() (femebe.Stream, error)
}

type Session interface {
	BackendKeyHolder
	Canceller
	Run() error
}

type SessionError struct {
	error
	Session Session
}

type simpleSessionManager struct {
	sessions []Session
	sessionLock sync.Mutex
}

func (s *simpleSessionManager) RunSession(session Session) error {
	s.sessionLock.Lock()
	s.sessions = append(s.sessions, session)
	s.sessionLock.Unlock()

	// N.B.: this is a blocking call that will not return until
	// the session completes
	err := session.Run()
	s.sessionLock.Lock()
	for i, si := range s.sessions {
		if si == session {
			// slice out this session
			copy(s.sessions[i:], s.sessions[i+1:])
			s.sessions[len(s.sessions)-1] = nil
			s.sessions = s.sessions[:len(s.sessions)-1]
			break
		}
	}
	s.sessionLock.Unlock()
	return err
}

func (s *simpleSessionManager) Cancel(backendPid, secretKey int32) error {
	for _, session := range s.sessions {
		// TODO: we could cache this info once available, but
		// for a reasonably small number of sessions, there's
		// probably no point
		if p, k := session.BackendKeyData(); p == backendPid && k == secretKey  {
			s.sessionLock.Lock()
			defer s.sessionLock.Lock()
			return session.Cancel(backendPid, secretKey)
		}
	}
	return errors.New("not found")
}

type SimpleConnector struct {
	backendAddr string
	startupMessage femebe.Message
	cancelMessage femebe.Message
}

func NewSimpleConnector(target string, options map[string]string) Connector {
	c := &SimpleConnector{backendAddr: target}
	message.InitStartupMessage(&c.startupMessage, options)
	return c
}

func (c *SimpleConnector) dial() (femebe.Stream, error) {
	conn, err := util.AutoDial(c.backendAddr)
	if err != nil {
		return nil, err
	}
	return femebe.NewBackendStream(conn), nil
}

func (c *SimpleConnector) Startup() (femebe.Stream, error) {
	beStream, err := c.dial()
	if err != nil {
		return nil, err
	}
	err = beStream.Send(&c.startupMessage)
	if err != nil {
		return nil, err
	}
	return beStream, nil
}

func (c *SimpleConnector) Cancel(backendPid, secretKey int32) error {
	beStream, err := c.dial()
	if err != nil {
		return err
	}
	message.InitCancelRequest(&c.cancelMessage, backendPid, secretKey)
	return beStream.Send(&c.cancelMessage)
}

type simpleRouter struct {
	backendPid int32
	backendKeyData int32
	from femebe.Stream
	to femebe.Stream
	feBuf femebe.Message
	beBuf femebe.Message
}

func NewSimpleRouter(from, to femebe.Stream) Router {
	return &simpleRouter{
		backendPid: -1,
		backendKeyData: -1,
		from: from,
		to: to,
	}
}

func (s *simpleRouter) BackendKeyData() (int32, int32) {
	return s.backendPid, s.backendKeyData
}

// route the next message from frontend to backend,
// blocking and flushing if necessary
func (s *simpleRouter) RouteFrontend() (err error) {
	err = s.from.Next(&s.feBuf)
	if err != nil {
		return
	}
	err = s.to.Send(&s.feBuf)
	if err != nil {
		return
	}
	if !s.from.HasNext() {
		return s.to.Flush()
	}
	return
}

// route the next message from backend to frotnend,
// blocking and flushing if necessary
func (s *simpleRouter) RouteBackend() error {
	err := s.from.Next(&s.beBuf)
	if err != nil {
		return err
	}
	if message.IsBackendKeyData(&s.beBuf) {
		beInfo, err := message.ReadBackendKeyData(&s.beBuf)
		if err != nil {
			return err
		}
		s.backendPid = beInfo.Pid
		s.backendKeyData = beInfo.Key
	}
	err = s.to.Send(&s.beBuf)
	if !s.from.HasNext() {
		return s.to.Flush()
	}
	return nil
}

type simpleSession struct {
	router Router
	Canceller
}

func NewSimpleSession(r Router, c Canceller) Session {
	return &simpleSession{r, c}
}

func (s *simpleSession) Run() (err error) {
	errs := make(chan error, 2)
	routeFrontend := func() { util.ErrToChannel(s.router.RouteFrontend, errs) }
	routeBackend := func() { util.ErrToChannel(s.router.RouteBackend, errs) }
	go routeFrontend()
	go routeBackend()
	err = <- errs
	// N.B.: we ignore the second error entirely, but we do wait
	// for it to ensure the session is fully cleaned up before we
	// exit
	_ = <- errs 
	return
}

func (s *simpleSession) BackendKeyData()  (int32, int32) {
	return s.router.BackendKeyData()
}

