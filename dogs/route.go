package main

type routingEntry struct {
	dbnameIn string
	addr string
	dbnameOut string
}

type routingTable struct {
	sync.RWMutex
	tab map[string] *routingEntry
}

func newRoutingTable() *routingTable {
	return &routingTable{
		sync.RWMutex, 
		make(map[string] *routingEntry)
	}
}

func (rt *routingTable) post(route *routingEntry) {
	rt.Lock()
	defer rt.Unlock()

	rt.tab[route.dbnameIn] = route
}

func (rt *routingTable) match(dbnameIn string) *routingTable {
	rt.RLock()
	defer rt.RUnlock()

	return rt.tab[route.dbnameIn]
}
