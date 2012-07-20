package dogconf


import (
	"fmt"
	"io"
	"strconv"
	"strings"
)


/*

 [route all [get]]

 [route all [delete]]

get info on given route:

 [route 'route-id' [get]]

add a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid'
   [create [addr='123.123.123.123:5432', user='foo', password='bar',dbname='x']]] 

patch a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid' @ 5
   [patch [lock='t']]] 

delete a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid' @ 5 [delete]]

*/

/*

grammar:

<request>    ::= "[" "route" <route-spec> "[" <command> "]" "]"
<route-spec> ::= "all" | <route-id>
<route-id>   ::= <identifier> "@" <ocn> | <identifier>
<command>    ::= <list-cmd> "[" <patch-list> "]" | <bare-cmd>
<bare-cmd>   ::= "get" | "delete"
<list-cmd>   ::= "patch" | "create"
<patch-list> ::= <patch> | <patch-list> "," <patch>
<patch>      ::= <identifier> "=" <value>
<value>      ::= <str-lit>
<ocn>        ::= <unsigned-integer>

*/

// Strip the quotes surrounding the string str and replace any escaped
// quotes by their actual values 
func stripStr(str string) (string) {
	// TODO: support quoted identifiers in addition to string literals
	l := len(str)
	if l < 2 || str[0] != '\'' || str[l - 1] != '\'' {
		panic(fmt.Sprintf("Malformed string lexeme: %v", str))
	}
	stripped := strings.Replace(str[1:len(str)-1], "''", "'", -1)
	return stripped
}

func expect(s *Scanner, tokTyp TokenType) (*Token, error) {
	tok := s.Scan()
	if tok.Type != tokTyp {
		return nil, fmt.Errorf("Expected token '%v'; got '%v'",
			TokenTypeStr(tokTyp), tok)
	}
	return tok, nil
}

func ParseRequest(r io.Reader) (RouteRequest, error) {
	var s = new(Scanner)
	s.Init(r)
	return parseRequest(s)
}

func parseRequest(s *Scanner) (RouteRequest, error) {
	_, err := expect(s, LBrace)
	if err != nil {
		return nil, err
	}
	tok, err := expect(s, Ident)
	if err != nil {
		return nil, err
	}
	if tok.Lexeme != "route" {
		return nil, fmt.Errorf("Expected 'route', got %v", tok)
	}
	spec, err := parseRouteSpec(s)
	if err != nil {
		return nil, err
	}
	_, err = expect(s, LBrace)
	if err != nil {
		return nil, err
	}
	cmd, err := parseCommand(spec, s)
	if err != nil {
		return nil, err
	}
	_, err = expect(s, RBrace)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func parseRouteSpec(s *Scanner) (*RouteSpec, error) {
	// Here we either expect the keyword/identifier 'all'
	// or a quoted database identifier, optionally with an ocn
	
	if tok := s.Peek(); tok.Type == Ident && tok.Lexeme == "all" {
		_, err := expect(s, Ident)
		if err != nil {
			return nil, err
		} else {
			return AllRoutes, nil
		}
	}

	tok, err := expect(s, String)
	if err != nil {
		return nil, err
	}	
	id := stripStr(tok.Lexeme)
	ocn := InvalidOcn
	// There may or may not be an OCN required for this command
	// and there may or may not be one present--we don't attempt
	// to resolve this at the grammar level. Note that we *never*
	// require an OCN for the 'all' commands.
	if tok := s.Peek(); tok.Type == At {
		_, err = expect(s, At)
		if err != nil {
			return nil, err
		}
		tok, err = expect(s, Int)
		if err != nil {
			return nil, err
		}
		ocnInt, err := strconv.ParseUint(tok.Lexeme, 10, 64)
		ocn = Ocn(ocnInt)
		if err != nil {
			return nil, err
		}		
	}
	return &RouteSpec{RouteId(id), Ocn(ocn)}, nil
}

func parseCommand(spec *RouteSpec, s *Scanner) (req RouteRequest, err error) {
	tok, err := expect(s, Ident)
	if err != nil {
		return nil, err
	}

	switch tok.Lexeme {
	case "patch", "create":
		_, err = expect(s, LBrace)
		if err != nil {
			return nil, err
		}
		patches, err := parsePatchList(s)
		if err != nil {
			return nil, err
		}
		_, err = expect(s, RBrace)
		if err != nil {
			return nil, err
		}
		if tok.Lexeme == "patch" {
			req = &PatchRequest{*spec, patches}
		} else {
			req = &AddRequest{*spec, patches}
		}
	case "get":
		req = &GetRequest{*spec}
	case "delete":
		// nothing to do here
		req = &DeleteRequest{*spec}
	default:
		return nil, fmt.Errorf("Expected 'patch', 'create', " +
			"'get', or 'delete'; got %v", tok)
	}
	return req, nil

}

func parsePatchList(s *Scanner) (map[string]string, error) {
	patchMap := make(map[string]string)
	allowComma := false
	for tok := s.Peek(); tok.Type != RBrace; tok = s.Peek() {
		if allowComma && tok.Type == Comma {
			_, err := expect(s, Comma)
			if err != nil {
				return nil, err
			}
		}
		keyTok, err := expect(s, Ident)
		if err != nil {
			return nil, err
		}
		_, err = expect(s, Equals)
		if err != nil {
			return nil, err
		}
		valTok, err := expect(s, String)
		if err != nil {
			return nil, err
		}
		switch k := keyTok.Lexeme; k {
		case "addr", "lock", "user", "password":
			_, present := patchMap[k]
			if !present {
				patchMap[k] = stripStr(valTok.Lexeme)
			} else {
				return nil, fmt.Errorf("Duplicate key '%v' " +
					" in patch request", keyTok)
			}
		default:
			return nil, fmt.Errorf("Unknown key '%v': expected " +
				"'addr', 'lock', 'user', or 'password'", keyTok)
		}
		
		allowComma = true
	}
	return patchMap, nil
}
