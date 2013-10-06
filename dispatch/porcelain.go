package dispatch

import (
	"crypto/tls"
	"errors"
	"fmt"
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
	Cancel(backendPid, secretKey uint32) error
}

type BackendKeyHolder interface {
	// Identifies this session, as per the FEBE protocol.
	// If this information is not yet available when this
	// method is called, or the router does not support
	// cancellation, it should return (0, 0).
	BackendKeyData() (uint32, uint32)
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
	Cancel(backendPid, secretKey uint32) error
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

func NewSimpleSessionManager() SessionManager {
	return &simpleSessionManager{}
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

func (s *simpleSessionManager) Cancel(backendPid, secretKey uint32) error {
	for _, session := range s.sessions {
		// TODO: we could cache this info once available, but
		// for a reasonably small number of sessions, there's
		// probably no point
		if p, k := session.BackendKeyData(); p == backendPid && k == secretKey  {
			s.sessionLock.Lock()
			defer s.sessionLock.Unlock()
			return session.Cancel(backendPid, secretKey)
		}
	}
	return errors.New("not found")
}

type simpleConnector struct {
	backendAddr string
	opts map[string]string
}

// Make a connector that always prefers TLS and connects using the
// options specified here.
func NewSimpleConnector(target string, options map[string]string) Connector {
	return &simpleConnector{backendAddr: target, opts: options}
}

func (c *simpleConnector) dial() (femebe.Stream, error) {
	bareConn, err := util.AutoDial(c.backendAddr)
	if err != nil {
		return nil, fmt.Errorf("could not connect to %v: %v", c.backendAddr, err)
	}

	// the simpleConnector always prefers TLS
	beConn, err := util.NegotiateTLS(bareConn, &util.SSLConfig{
		Mode: util.SSLPrefer,
		Config: tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		return nil, fmt.Errorf("could not negotiate TLS: %v", err)
	}

	return femebe.NewBackendStream(beConn), nil
}

func (c *simpleConnector) Startup() (femebe.Stream, error) {
	beStream, err := c.dial()
	if err != nil {
		return nil, err
	}
	var startup femebe.Message
	message.InitStartupMessage(&startup, c.opts)
	err = beStream.Send(&startup)
	if err != nil {
		return nil, err
	}
	return beStream, nil
}

func (c *simpleConnector) Cancel(backendPid, secretKey uint32) error {
	beStream, err := c.dial()
	defer beStream.Close()
	if err != nil {
		return err
	}
	var cancel femebe.Message
	message.InitCancelRequest(&cancel, backendPid, secretKey)
	return beStream.Send(&cancel)
}

type simpleRouter struct {
	backendPid uint32
	secretKey uint32
	fe femebe.Stream
	be femebe.Stream
	feBuf femebe.Message
	beBuf femebe.Message
}

func NewSimpleRouter(fe, be femebe.Stream) Router {
	return &simpleRouter{
		backendPid: 0,
		secretKey: 0,
		fe: fe,
		be: be,
	}
}

func (s *simpleRouter) BackendKeyData() (uint32, uint32) {
	return s.backendPid, s.secretKey
}

// route the next message from frontend to backend,
// blocking and flushing if necessary
func (s *simpleRouter) RouteFrontend() (err error) {
	err = s.fe.Next(&s.feBuf)
	if err != nil {
		return
	}
	err = s.be.Send(&s.feBuf)
	if err != nil {
		return
	}
	if !s.fe.HasNext() {
		return s.be.Flush()
	}
	return
}

// route the next message from backend to frotnend,
// blocking and flushing if necessary
func (s *simpleRouter) RouteBackend() error {
	err := s.be.Next(&s.beBuf)
	if err != nil {
		return err
	}
	if message.IsBackendKeyData(&s.beBuf) {
		beInfo, err := message.ReadBackendKeyData(&s.beBuf)
		if err != nil {
			return err
		}
		s.backendPid = beInfo.BackendPid
		s.secretKey = beInfo.SecretKey
	}
	err = s.fe.Send(&s.beBuf)
	if !s.be.HasNext() {
		return s.fe.Flush()
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

func (s *simpleSession) BackendKeyData()  (uint32, uint32) {
	return s.router.BackendKeyData()
}

