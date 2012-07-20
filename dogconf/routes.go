package dogconf

import (
	"sync"
	"net"
)

type RouteId string
const InvalidRoute = ""

type Ocn uint64
const InvalidOcn = Ocn(0)
const FirstOcn = 1

type RouteSpec struct {
	Route RouteId
	Ocn Ocn
}

var AllRoutes = &RouteSpec{InvalidRoute, InvalidOcn}

// Shared interface for all the request types
type RouteRequest interface {
	Process(routes *RouteMap, essions *SessionMap) (RouteResult, error)
}

type QueryResult struct {
	Columns []string
	Data [][]string
}

type RouteResult interface {
	Encode() QueryResult
}



type RouteChangeResult struct {
	RouteSpec
}

type RouteQueryResult struct {
	RouteSpec
	Properties map[string]string
}

// Return [ NextOcn ]
type PatchRequest struct {
	RouteSpec
	Patches map[string]string
}

// Return [ NextOcn ]
type AddRequest struct {
	RouteSpec
	Values map[string]string
}

// Return [ NextOcn, RouteId, PropMap ]
type GetRequest struct {
	RouteSpec
}

// Return [ NULL ]
type DeleteRequest struct {
	RouteSpec
}

type RouteInfo struct {
	Ocn Ocn
	Id RouteId
	Addr string
	Lock string
	User string
	Password string
}

type RouteMap struct {
	mapping map[RouteId] *RouteInfo
	lock sync.RWMutex
}

func NewRouteMap() *RouteMap {
	m := make(map[RouteId] *RouteInfo)
	var l sync.RWMutex
	return &RouteMap{m, l}
}

func (m *RouteMap) Lock() {
	m.lock.Lock()
}

func (m *RouteMap) RLock() {
	m.lock.RLock()
}

func (m *RouteMap) Unlock() {
	m.lock.Unlock()
}

func (m *RouteMap) RUnlock() {
	m.lock.RUnlock()
}

type SessionInfo struct {
	Source string
	Connection net.Conn // ???
	RouteData RouteInfo
}

type SessionMap struct {
	mapping map[RouteId] *SessionInfo
	lock sync.RWMutex
}

func NewSessionMap() *SessionMap {
	m := make(map[RouteId] *SessionInfo)
	var l sync.RWMutex
	return &SessionMap{m, l}
}

func (m *SessionMap) Lock() {
	m.lock.Lock()
}

func (m *SessionMap) RLock() {
	m.lock.RLock()
}

func (m *SessionMap) Unlock() {
	m.lock.Unlock()
}

func (m *SessionMap) RUnlock() {
	m.lock.RUnlock()
}

// right abstraction?

