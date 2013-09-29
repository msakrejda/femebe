package femebe

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
	BackendKeyData() int32, int32
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
	Resolve(protoVersion int, params map[string]string) Connector
}

type Canceller interface {
	// Open a stream to a backend and send a cancellation
	// request with the given data, the close the stream.
	Cancel(backendPid, secretKey int32) error
}

type Connector interface {
	Canceller
	// Open a stream to a backend and send a StartupMessage on
	// that stream before returning it. Return an error if a stream
	// cannot be established or if sending the startup packet
	// returns an error
	Startup() beStream, error
}

type Session interface {
	BackendKeyHolder
	Cancel(int32, int32) error
	Run() error
}

// so...
func handle(conn net.Conn) {
	feStream := NewFEStream(conn)
	var m Message
	err := feStream.Next(m)
	if err != nil {
		// ...
	}
	if message.IsStartup(m) {
		startup, err := message.ReadStartup(&m)
		if err != nil {
			// ... 
		}
		connector, err := resolver.Resolve(m.Version, m.Options)
		if err != nil {
			// ...
		}
		beStream, err := connector.Startup()
		if err != nil {
			// ...
		}

		router := femebe.NewSimpleRouter(feStream, beStream)
		session := femebe.NewSimpleSession(router, connector)

		go manager.RunSession(session)

	} else if message.IsCancel(m) {
		cancel, err := message.ReadCancel(&m)
		if err != nil {
			// ... 
		}
		go manager.Cancel(cancel.BackendPid, cancel.SecretKey)
	} else {
		// unknown message type: we can't do anything with this
		_ = conn.Close()
	}
}

type SessionError struct {
	error
	Session Session
}

type simpleSessionManager struct {
	sessions []Session
}

// and...
func (s *simpleSessionManager) RunSession(session Session) error {
	s.sessionLock.Lock()
	s.sessions = append(s.sessions, session)
	s.sessionLock.Unlock()

	// N.B.: this is a blocking call that will not return until
	// the session completes
	return session.Run()
}

func (s *simpleSessionManager) Cancel(backendPid, secretKey int32) error {
	found := false
	for _, session := range c.sessions {
		// TODO: we could cache this info once available, but
		// for a reasonably small number of sessions, there's
		// probably no point
		if pid, keyData == session.BackendKeyData() {
			s.sessionLock.Lock()
			err := session.Cancel(backendPid, secretKey)
			if err != nil {
				s.errors <- err
			}
			s.sessionLock.Lock()
			found = true
			break
		}
	}
	if !found {
		return errors.New("not found")
	}
}

type SimpleConnector struct {
	backendAddr string
	startupMessage femebe.Message
	cancelMessage femebe.Message
}

func NewSimpleConnector(target string, options map[string]string) Connector, error {
	c := &SimpleConnector{backendAddr: target}
	message.InitStartup(&c.startupMessage)
	return c
}

func (c *SimpleConnector) dial() Stream, error {
	conn, err := net.Dial(backendAddr)
	if err != nil {
		return err
	}
	return NewBackendStream(conn)
}

func (c *SimpleConnector) Startup() Stream, error {
	beStream, err := c.dial()
	if err != nil {
		return nil, err
	}
	err := beStream.Send(&c.startupMessage)
	if err != nil {
		return nil, err
	}
	return beStream, nil
}

func (c *SimpleConnector) Cancel(backendPid, secreteKey int32) error {
	beStream, err := c.dial()
	if err != nil {
		return err
	}
	err := femebe.InitCancel(&c.cancelMessage, backendPid, secretKey)
	if err != nil {
		return err
	}
	err = beStream.Send(&c.cancelMessage)
	if err != nil {
		return err
	}
	
}

type simpleRouter struct {
	backendPid int32
	backendKeyData int32
	from Stream
	to Stream
	feBuf Message
	beBuf Message
}

func NewSimpleRouter(from, to Stream) Router {
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
	err = from.Next(&s.feBuf)
	if err != nil {
		return
	}
	err = to.Send(&s.feBuf)
	if err != nil {
		return
	}
	if !from.HasNext() {
		return to.Flush()
	}
	return
}

// route the next message from backend to frotnend,
// blocking and flushing if necessary
func (s *simpleRouter) RouteBackend() error {
	err = from.Next(&s.beBuf)
	if err != nil {
		return
	}
	if message.IsBackendKeyData(&s.beBuf) {
		beInfo, err := message.ReadBackendKeyData(&s.beBuf)
		if err != nil {
			return
		}
		s.BackendPid = beInfo.BackendPid
		s.BackendKeyData = beInfo.BackendKeyData
	}
	err = to.Send(&m)
	if !from.HasNext() {
		return to.Flush()
	}
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
	routeFrontend := func() { errToChannel(s.router.routeFrotend, errs) }
	routeBackend := func() { errToChannel(s.router.routeBackend, errs) }
	go routeFrontend()
	go routeBackend()
	err = <- errs
	// N.B.: we ignore the second error entirely, but we do wait
	// for it to ensure the session is fully cleaned up before we
	// exit
	_ = <- errs 
	return
}
