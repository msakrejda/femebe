package main

import (
	"femebe/pgproto"
	"sync"
)

type routingEntry struct {
	dbnameIn  string
	addr      string
	dbnameOut string
}

type routingTable struct {
	sync.RWMutex
	tab map[string]*routingEntry
}

func newRoutingTable() *routingTable {
	return &routingTable{
		sync.RWMutex{},
		make(map[string]*routingEntry),
	}
}

func (rt *routingTable) post(route *routingEntry) {
	rt.Lock()
	defer rt.Unlock()

	rt.tab[route.dbnameIn] = route
}

func (rt *routingTable) match(dbnameIn string) *routingEntry {
	rt.RLock()
	defer rt.RUnlock()

	return rt.tab[dbnameIn]
}

func (rt *routingTable) rewrite(s *pgproto.Startup) (route *routingEntry) {
	route = rt.match(s.Params["database"])
	if route != nil {
		s.Params["database"] = route.dbnameOut
	}

	return route
}
