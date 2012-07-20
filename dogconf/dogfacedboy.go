package dogconf

import (
	"io"
	"fmt"
)

/*

 ~doglang~

dump all routes:

 [route all [get]]

get info on given route:

 [route 'route-id' [get]]

add a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid'
   [create [addr='123.123.123.123:5432', user='foo', password='bar']]] 

patch a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid' @ 5
   [patch [lock='t']]] 

delete a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid' @ 5 [delete]]
 
*/


func HandleRequest(routes *RouteMap, sessions *SessionMap, reqReader io.Reader) (error) {
	req, err := ParseRequest(reqReader)
	if err != nil {
		return err
	}
	
	result, err := ProcessRequest(routes, sessions, req)
}


func ProcessRequest(routes *RouteMap, sessions *SessionMap, req RouteRequest) (error) {
	// TODO: better way of dealing with the 'all' paths; return values
	// perhaps these should just be done via a Process method on the
	// various RouteRequests
	switch req.(type) {
	case *AddRequest:
		addReq := req.(*AddRequest)
		routeId := addReq.Route
		if routeId == InvalidRoute {
			return fmt.Errorf("Add request not applicable to all routes")
		}
		if addReq.Ocn != InvalidOcn {
			return fmt.Errorf("OCN not applicable to add transaction")
		}

		routes.Lock()
		defer routes.Unlock()

		if _, ok := routes.mapping[routeId]; ok {
			return fmt.Errorf("Route with identifier %v already exists",
				addReq.Route)
		}
		addr := addReq.Values["addr"]
		lock := addReq.Values["lock"]
		user := addReq.Values["user"]
		password := addReq.Values["password"]

		info := &RouteInfo{FirstOcn, routeId, addr, lock, user, password}

		routes.mapping[routeId] = info

		return fmt.Errorf("No error on add")

	case *PatchRequest:
		patchReq := req.(*PatchRequest)
		routeId := patchReq.Route
		if routeId == InvalidRoute {
			return fmt.Errorf("Patch request not applicable to all routes")
		}
		if patchReq.Ocn == InvalidOcn {
			return fmt.Errorf("OCN required for route patching")
		}
		
		routes.Lock()
		defer routes.Unlock()

		info, ok := routes.mapping[routeId]
		if !ok {
			return fmt.Errorf("No route with identifier %v exists",
				patchReq.Route)
		}

		if patchReq.Ocn != info.Ocn {
			return fmt.Errorf("OCN mismatch; expected %v")
		}

		info.Addr = patchReq.Patches["addr"]
		info.Lock = patchReq.Patches["lock"]
		info.User = patchReq.Patches["user"]
		info.Password = patchReq.Patches["password"]

		info.Ocn += 1

		return fmt.Errorf("No error on patch")

	case *GetRequest:

	case *DeleteRequest:

	default:
		return fmt.Errorf("Unknown request %v", req)
	}
	return nil
}


