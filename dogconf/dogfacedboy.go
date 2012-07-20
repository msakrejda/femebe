package dogconf

import (
	"io"
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

func HandleRequest(routes *RouteMap, sessions *SessionMap, reqReader io.Reader) (RouteResult, error) {
	req, err := ParseRequest(reqReader)
	if err != nil {
		// or send ErrorResponse
		return nil, err
	}

	return req.Process(routes, sessions)

	// get columns and data from result, send as response
	//    OR
	// get error, send as ErrorResponse
}
