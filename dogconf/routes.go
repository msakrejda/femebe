package dogconf

import (
	"fmt"
	"net"
	"strconv"
	"sync"
)

type RouteId string

const InvalidRoute = ""

type Ocn uint64

const InvalidOcn = Ocn(0)
const FirstOcn = 1

type RouteSpec struct {
	Route RouteId
	Ocn   Ocn
}

var AllRoutes = &RouteSpec{InvalidRoute, InvalidOcn}

// Shared interface for all the request types
type RouteRequest interface {
	Process(routes *RouteMap, essions *SessionMap) (RouteResult, error)
}

type RouteResult interface {
	GetFields() []string
	GetData() [][]string
}

type RouteChangeResult struct {
	routeId RouteId
	ocn     Ocn
}

func (r *RouteChangeResult) GetFields() []string {
	return []string{"RouteId", "Ocn"}
}

func (r *RouteChangeResult) GetData() [][]string {
	idStr := string(r.routeId)
	ocnStr := strconv.FormatUint(uint64(r.ocn), 10)
	return [][]string{{idStr, ocnStr}}
}

type RouteQueryResult struct {
	data []*RouteInfo
}

func (r *RouteQueryResult) GetFields() []string {
	return []string{"RouteId", "Ocn", "Addr", "Lock",
		"User", "Password"}
}

func (r *RouteQueryResult) GetData() [][]string {
	result := make([][]string, len(r.data))
	for i, info := range r.data {
		result[i] = []string{
			string(info.Id),
			strconv.FormatUint(uint64(info.Ocn), 10),
			info.Addr,
			info.Lock,
			info.User,
			info.Password,
		}
	}
	return result
}

// Returns [ NextOcn, RouteId ]
type PatchRequest struct {
	RouteSpec
	Patches map[string]string
}

func (r *PatchRequest) Process(routes *RouteMap, sessions *SessionMap) (RouteResult, error) {
	routeId := r.Route
	if routeId == InvalidRoute {
		return nil, fmt.Errorf("Patch request not applicable to all routes")
	}
	if r.Ocn == InvalidOcn {
		return nil, fmt.Errorf("OCN required for route patching")
	}

	routes.Lock()
	defer routes.Unlock()

	info, ok := routes.mapping[routeId]
	if !ok {
		return nil, fmt.Errorf("No route with identifier %v exists",
			r.Route)
	}

	if r.Ocn != info.Ocn {
		return nil, fmt.Errorf("OCN mismatch; expected %v")
	}

	info.Addr = r.Patches["addr"]
	info.Lock = r.Patches["lock"]
	info.User = r.Patches["user"]
	info.Password = r.Patches["password"]

	info.Ocn += 1

	return &RouteChangeResult{info.Id, info.Ocn}, nil
}

// Returns [ NextOcn, RouteId ]
type AddRequest struct {
	RouteSpec
	Values map[string]string
}

func (r *AddRequest) Process(routes *RouteMap, session *SessionMap) (RouteResult, error) {
	routeId := r.Route
	if routeId == InvalidRoute {
		return nil, fmt.Errorf("Add request not applicable to all routes")
	}
	if r.Ocn != InvalidOcn {
		return nil, fmt.Errorf("OCN not applicable to add request")
	}

	routes.Lock()
	defer routes.Unlock()

	if _, ok := routes.mapping[routeId]; ok {
		return nil, fmt.Errorf("Route with identifier %v already exists",
			r.Route)
	}
	addr := r.Values["addr"]
	lock := r.Values["lock"]
	user := r.Values["user"]
	password := r.Values["password"]

	info := &RouteInfo{FirstOcn, routeId, addr, lock, user, password}

	routes.mapping[routeId] = info

	return &RouteChangeResult{info.Id, info.Ocn}, nil
}

// Returns [ NextOcn, RouteId, Addr, Lock, User, Password ]
type GetRequest struct {
	RouteSpec
}

func (r *GetRequest) Process(routes *RouteMap, session *SessionMap) (RouteResult, error) {
	routeId := r.Route

	if r.Ocn != InvalidOcn {
		return nil, fmt.Errorf("OCN not applicable to get request")
	}

	routes.RLock()
	defer routes.RUnlock()

	var result RouteResult
	if routeId == InvalidRoute {
		infos := make([]*RouteInfo, len(routes.mapping))
		// iterate over all route ids
		i := 0
		for _, info := range routes.mapping {
			infos[i] = info
			i++
		}
		result = &RouteQueryResult{infos}
	} else {
		routeInfo, ok := routes.mapping[routeId]
		if !ok {
			return nil, fmt.Errorf("Route id %v not found", routeId)
		}
		result = &RouteQueryResult{[]*RouteInfo{routeInfo}}
	}

	return result, nil
}

// Returns [ NextOcn, RouteId]
type DeleteRequest struct {
	RouteSpec
}

func (r *DeleteRequest) Process(routes *RouteMap, session *SessionMap) (RouteResult, error) {
	return nil, nil
}

type RouteInfo struct {
	Ocn      Ocn
	Id       RouteId
	Addr     string
	Lock     string
	User     string
	Password string
}

type RouteMap struct {
	mapping map[RouteId]*RouteInfo
	lock    sync.RWMutex
}

func NewRouteMap() *RouteMap {
	m := make(map[RouteId]*RouteInfo)
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

type BackendKeyData struct {
	Pid    int32
	Secret int32
}

type SessionInfo struct {
	Source     string
	KeyData    BackendKeyData
	Connection net.Conn  // ???
	RouteData  RouteInfo // ???
}

type SessionMap struct {
	mapping map[RouteId]*SessionInfo
	lock    sync.RWMutex
}

func NewSessionMap() *SessionMap {
	m := make(map[RouteId]*SessionInfo)
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
