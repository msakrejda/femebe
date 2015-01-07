package femebe

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/uhoh-itsmaciek/femebe/core"
	"github.com/uhoh-itsmaciek/femebe/proto"
	"github.com/uhoh-itsmaciek/femebe/util"
	"sync"
)

// SessionManager is responsible for tracking all the currently
// running sessions and passing on any cancellation requests
type SessionManager interface {
	RunSession(session Session) error
	Cancel(backendPid, secretKey uint32) error
}

// BackendKeyHolder holds cancellation data for a particular
// connection
type BackendKeyHolder interface {
	// Identifies this session, as per the FEBE protocol.
	// If this information is not yet available when this
	// method is called, or the router does not support
	// cancellation, it should return (0, 0).
	BackendKeyData() (uint32, uint32)
}

// Router moves protocol messages from the frontend to the backend
// (and vice versa). It is also responsible for exposing cancellation
// key data.
type Router interface {
	BackendKeyHolder
	// Route the next message from the frontend
	RouteFrontend() error
	// Route the next message from the bcakend
	RouteBackend() error
}

// Resolver resolves the given startup message parameters into a
// Connector to a given backend.
type Resolver interface {
	// Resolve a given startup message into a Connector that can be
	// used to connect to or send cancellations to the given backend
	Resolve(params map[string]string) Connector
}

// Canceller can send a Postgres CancelRequest message to the backend
// (or delegate it). Should return an error if it knows that the
// request will not succeed.
//
// Note that due to the nature of the cancellation mechanism, there is
// no guarantee of success, so the absence of an error does not
// necessarily mean success.
type Canceller interface {
	// Open a stream to a backend and send a cancellation
	// request with the given data, the close the stream.
	Cancel(backendPid, secretKey uint32) error
}

// Connector knows how to reach a single backend for the purpose of
// resolving a connection. Typically this is only used for starting a
// fresh connection, but every query cancellation also uses this
// mechanism.
type Connector interface {
	Canceller
	// Open a stream to a backend, go through TLS negotiation (if
	// desired), and send a StartupMessage on that stream before
	// returning it. Return an error if a stream cannot be
	// established or if sending the startup packet returns an
	// error
	Startup() (core.Stream, error)
}

// Session represents a single client-server connection.
type Session interface {
	BackendKeyHolder
	Canceller
	// Run the session until completion, relaying frontend and
	// backend messages, and return the error, if any.
	Run() error
}

type simpleSessionManager struct {
	sessions    []Session
	sessionLock sync.Mutex
}

// Return the default SessionManager, with bookkeeping for
// cancellation.
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
		if p, k := session.BackendKeyData(); p == backendPid && k == secretKey {
			s.sessionLock.Lock()
			defer s.sessionLock.Unlock()
			return session.Cancel(backendPid, secretKey)
		}
	}
	return errors.New("not found")
}

type simpleConnector struct {
	backendAddr string
	opts        map[string]string
}

// Make a Connector that always prefers TLS and connects using the
// options specified here.
func NewSimpleConnector(target string, options map[string]string) Connector {
	return &simpleConnector{backendAddr: target, opts: options}
}

func (c *simpleConnector) dial() (core.Stream, error) {
	bareConn, err := util.AutoDial(c.backendAddr)
	if err != nil {
		return nil, fmt.Errorf("could not connect to %v: %v", c.backendAddr, err)
	}

	// the simpleConnector always prefers TLS
	beConn, err := util.NegotiateTLS(bareConn, &util.SSLConfig{
		Mode:   util.SSLPrefer,
		Config: tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		return nil, fmt.Errorf("could not negotiate TLS: %v", err)
	}

	return core.NewBackendStream(beConn), nil
}

func (c *simpleConnector) Startup() (core.Stream, error) {
	beStream, err := c.dial()
	if err != nil {
		return nil, err
	}
	var startup core.Message
	proto.InitStartupMessage(&startup, c.opts)
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
	var cancel core.Message
	proto.InitCancelRequest(&cancel, backendPid, secretKey)
	return beStream.Send(&cancel)
}

type simpleRouter struct {
	backendPid uint32
	secretKey  uint32
	fe         core.Stream
	be         core.Stream
	feBuf      core.Message
	beBuf      core.Message
}

// Make a new Router that captures cancellation data and ferries
// messages back and forth for the two streams. Flush the "to" stream
// when no more messages are available on the "from" stream, in both
// directions.
func NewSimpleRouter(fe, be core.Stream) Router {
	return &simpleRouter{
		backendPid: 0,
		secretKey:  0,
		fe:         fe,
		be:         be,
	}
}

func (s *simpleRouter) BackendKeyData() (uint32, uint32) {
	return s.backendPid, s.secretKey
}

func (s *simpleRouter) RouteFrontend() (err error) {
	// route the next message from frontend to backend,
	// blocking and flushing if necessary
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

func (s *simpleRouter) RouteBackend() error {
	// route the next message from backend to frotnend,
	// blocking and flushing if necessary
	err := s.be.Next(&s.beBuf)
	if err != nil {
		return err
	}
	if proto.IsBackendKeyData(&s.beBuf) {
		beInfo, err := proto.ReadBackendKeyData(&s.beBuf)
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

// Make a new Session that drives the given router and uses its
// cancellation data to delegate cancellation requests.
func NewSimpleSession(r Router, c Canceller) Session {
	return &simpleSession{r, c}
}

func (s *simpleSession) Run() (err error) {
	errs := make(chan error, 2)
	routeFrontend := func() { util.ErrToChannel(s.router.RouteFrontend, errs) }
	routeBackend := func() { util.ErrToChannel(s.router.RouteBackend, errs) }
	go routeFrontend()
	go routeBackend()
	err = <-errs
	// N.B.: we ignore the second error entirely, but we do wait
	// for it to ensure the session is fully cleaned up before we
	// exit
	_ = <-errs
	return
}

func (s *simpleSession) BackendKeyData() (uint32, uint32) {
	return s.router.BackendKeyData()
}
