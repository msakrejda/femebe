package codec

import (
	"github.com/uhoh-itsmaciek/femebe/proto"
)

// GuessOids attemps to guess the Postgres oids for the given data
// values. It assumes that rows is a slice of uniform-length slices
// where each cell corresponds to a column. It returns a slice of oid
// values of the mapped oids, or OID_UNKNOWN where no mapping could
// be determined.
func GuessOids(rows [][]interface{}) (oids []proto.Oid) {
	if len(rows) == 0 {
		// can't really make much of a guess here
		return []proto.Oid{}
	}
	oids = make([]proto.Oid, len(rows[0]))
	for _, row := range rows {
		gotAll := true
		for i := range oids {
			if o := oids[i]; o == 0 || o == proto.OidUnknown {
				oids[i] = MappedOid(row[i])
				if oids[i] == proto.OidUnknown {
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

// Mappedproto.Oid returns the Postgres oid mapped to the type of the given
// value in femebe, or OID_UNKNOWN if no mapping exists.
func MappedOid(val interface{}) proto.Oid {
	switch val.(type) {
	case nil:
		// we can't determine a type here
		return proto.OidUnknown
	case int16:
		return proto.OidInt2
	case int32:
		return proto.OidInt4
	case int64:
		return proto.OidInt8
	case float32:
		return proto.OidFloat4
	case float64:
		return proto.OidFloat8
	case string:
		return proto.OidText
	case bool:
		return proto.OidBool
	default:
		return proto.OidUnknown
	}
}
