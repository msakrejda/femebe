package codec

import (
	"github.com/deafbybeheading/femebe/pgproto"
)

// GuessOids attemps to guess the Postgres oids for the given data
// values. It assumes that rows is a slice of uniform-length slices
// where each cell corresponds to a column. It returns a slice of oid
// values of the mapped oids, or OID_UNKNOWN where no mapping could
// be determined.
func GuessOids(rows [][]interface{}) (oids []pgproto.Oid) {
	if len(rows) == 0 {
		// can't really make much of a guess here
		return []pgproto.Oid{}
	}
	oids = make([]pgproto.Oid, len(rows[0]))
	for _, row := range rows {
		gotAll := true
		for i, _ := range oids {
			if o := oids[i]; o == 0 || o == pgproto.OidUnknown {
				oids[i] = MappedOid(row[i])
				if oids[i] == pgproto.OidUnknown {
					gotAll = false
				}
			}
		}
		if gotAll {
			break
		}
	}
	return oids
}

// Mappedpgproto.Oid returns the Postgres oid mapped to the type of the given
// value in femebe, or OID_UNKNOWN if no mapping exists.
func MappedOid(val interface{}) pgproto.Oid {
	switch val.(type) {
	case nil:
		// we can't determine a type here
		return pgproto.OidUnknown
	case int16:
		return pgproto.OidInt2
	case int32:
		return pgproto.OidInt4
	case int64:
		return pgproto.OidInt8
	case float32:
		return pgproto.OidFloat4
	case float64:
		return pgproto.OidFloat8
	case string:
		return pgproto.OidText
	case bool:
		return pgproto.OidBool
	default:
		return pgproto.OidUnknown
	}

	panic("Oh snap!")
}
