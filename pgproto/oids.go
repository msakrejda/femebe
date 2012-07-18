package pgproto

// GuessOids attemps to guess the Postgres oids for the given data
// values. It assumes that rows is a slice of uniform-length slices
// where each cell corresponds to a column. It returns a slice of oid
// values of the mapped oids, or OID_UNKNOWN where no mapping could
// be determined.
func GuessOids(rows [][]interface{}) (oids []uint32) {
	if len(rows) == 0 {
		// can;t really make much of a guess here
		return []uint32{}
	}
	oids = make([]uint32, len(rows[0]))
	for _, row := range rows {
		gotAll := true
		for i, _ := range oids {
			if o := oids[i]; o == 0 || o == OID_UNKNOWN {
				oids[i] = MappedOid(row[i])
				if oids[i] == OID_UNKNOWN {
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

// MappedOid returns the Postgres oid mapped to the type of the given
// value in femebe, or OID_UNKNOWN if no mapping exists.
func MappedOid(val interface{}) uint32 {
	switch val.(type) {
	case nil:
		// we can't determine a type here
		return OID_UNKNOWN
	case int16:
		return OID_INT2
	case int32:
		return OID_INT4
	case int64:
		return OID_INT8
	case float32:
		return OID_FLOAT4
	case float64:
		return OID_FLOAT8
	case string:
		return OID_TEXT
	case bool:
		return OID_BOOL
	default:
		return OID_UNKNOWN
	}
	panic("Oh snap!")
}

// TypSize returns the size in bytes of the Postgres type specified by
// typOid, where undertood by femebe. For variable-length types or if
// the type is not known, -1 is returned.
func TypSize(typOid uint32) int16 {
	// TODO: right now, we hardcode the length of the various types
	// here; ideally, we should have a mapping for the fixed-length
	// types (although it seems that all the dynamic-length types,
	// even those with explicit length restrictions, is just -1, so
	// perhaps we can provide a mapping for that as well).
	switch typOid {
	case OID_BOOL:
		return 1
	case OID_INT2:
		return 2
	case OID_INT4, OID_FLOAT4:
		return 4
	case OID_INT8, OID_FLOAT8:
		return 8
	case OID_TEXT:
		return -1
	default:
		// unknown, assume variable length
		return -1
	}
}

// The oids of the Postgres built-in types
const (
	// generated via
	// psql -qAt -F $'\t' -p 5434 postgres -c
	//   "select 'OID_' || upper(typname), '=' || oid from pg_type"
	OID_BOOL                                  uint32 = 16
	OID_BYTEA                                        = 17
	OID_CHAR                                         = 18
	OID_NAME                                         = 19
	OID_INT8                                         = 20
	OID_INT2                                         = 21
	OID_INT2VECTOR                                   = 22
	OID_INT4                                         = 23
	OID_REGPROC                                      = 24
	OID_TEXT                                         = 25
	OID_OID                                          = 26
	OID_TID                                          = 27
	OID_XID                                          = 28
	OID_CID                                          = 29
	OID_OIDVECTOR                                    = 30
	OID_PG_TYPE                                      = 71
	OID_PG_ATTRIBUTE                                 = 75
	OID_PG_PROC                                      = 81
	OID_PG_CLASS                                     = 83
	OID_JSON                                         = 114
	OID_XML                                          = 142
	OID__XML                                         = 143
	OID__JSON                                        = 199
	OID_PG_NODE_TREE                                 = 194
	OID_SMGR                                         = 210
	OID_POINT                                        = 600
	OID_LSEG                                         = 601
	OID_PATH                                         = 602
	OID_BOX                                          = 603
	OID_POLYGON                                      = 604
	OID_LINE                                         = 628
	OID__LINE                                        = 629
	OID_FLOAT4                                       = 700
	OID_FLOAT8                                       = 701
	OID_ABSTIME                                      = 702
	OID_RELTIME                                      = 703
	OID_TINTERVAL                                    = 704
	OID_UNKNOWN                                      = 705
	OID_CIRCLE                                       = 718
	OID__CIRCLE                                      = 719
	OID_MONEY                                        = 790
	OID__MONEY                                       = 791
	OID_MACADDR                                      = 829
	OID_INET                                         = 869
	OID_CIDR                                         = 650
	OID__BOOL                                        = 1000
	OID__BYTEA                                       = 1001
	OID__CHAR                                        = 1002
	OID__NAME                                        = 1003
	OID__INT2                                        = 1005
	OID__INT2VECTOR                                  = 1006
	OID__INT4                                        = 1007
	OID__REGPROC                                     = 1008
	OID__TEXT                                        = 1009
	OID__OID                                         = 1028
	OID__TID                                         = 1010
	OID__XID                                         = 1011
	OID__CID                                         = 1012
	OID__OIDVECTOR                                   = 1013
	OID__BPCHAR                                      = 1014
	OID__VARCHAR                                     = 1015
	OID__INT8                                        = 1016
	OID__POINT                                       = 1017
	OID__LSEG                                        = 1018
	OID__PATH                                        = 1019
	OID__BOX                                         = 1020
	OID__FLOAT4                                      = 1021
	OID__FLOAT8                                      = 1022
	OID__ABSTIME                                     = 1023
	OID__RELTIME                                     = 1024
	OID__TINTERVAL                                   = 1025
	OID__POLYGON                                     = 1027
	OID_ACLITEM                                      = 1033
	OID__ACLITEM                                     = 1034
	OID__MACADDR                                     = 1040
	OID__INET                                        = 1041
	OID__CIDR                                        = 651
	OID__CSTRING                                     = 1263
	OID_BPCHAR                                       = 1042
	OID_VARCHAR                                      = 1043
	OID_DATE                                         = 1082
	OID_TIME                                         = 1083
	OID_TIMESTAMP                                    = 1114
	OID__TIMESTAMP                                   = 1115
	OID__DATE                                        = 1182
	OID__TIME                                        = 1183
	OID_TIMESTAMPTZ                                  = 1184
	OID__TIMESTAMPTZ                                 = 1185
	OID_INTERVAL                                     = 1186
	OID__INTERVAL                                    = 1187
	OID__NUMERIC                                     = 1231
	OID_TIMETZ                                       = 1266
	OID__TIMETZ                                      = 1270
	OID_BIT                                          = 1560
	OID__BIT                                         = 1561
	OID_VARBIT                                       = 1562
	OID__VARBIT                                      = 1563
	OID_NUMERIC                                      = 1700
	OID_REFCURSOR                                    = 1790
	OID__REFCURSOR                                   = 2201
	OID_REGPROCEDURE                                 = 2202
	OID_REGOPER                                      = 2203
	OID_REGOPERATOR                                  = 2204
	OID_REGCLASS                                     = 2205
	OID_REGTYPE                                      = 2206
	OID__REGPROCEDURE                                = 2207
	OID__REGOPER                                     = 2208
	OID__REGOPERATOR                                 = 2209
	OID__REGCLASS                                    = 2210
	OID__REGTYPE                                     = 2211
	OID_UUID                                         = 2950
	OID__UUID                                        = 2951
	OID_TSVECTOR                                     = 3614
	OID_GTSVECTOR                                    = 3642
	OID_TSQUERY                                      = 3615
	OID_REGCONFIG                                    = 3734
	OID_REGDICTIONARY                                = 3769
	OID__TSVECTOR                                    = 3643
	OID__GTSVECTOR                                   = 3644
	OID__TSQUERY                                     = 3645
	OID__REGCONFIG                                   = 3735
	OID__REGDICTIONARY                               = 3770
	OID_TXID_SNAPSHOT                                = 2970
	OID__TXID_SNAPSHOT                               = 2949
	OID_INT4RANGE                                    = 3904
	OID__INT4RANGE                                   = 3905
	OID_NUMRANGE                                     = 3906
	OID__NUMRANGE                                    = 3907
	OID_TSRANGE                                      = 3908
	OID__TSRANGE                                     = 3909
	OID_TSTZRANGE                                    = 3910
	OID__TSTZRANGE                                   = 3911
	OID_DATERANGE                                    = 3912
	OID__DATERANGE                                   = 3913
	OID_INT8RANGE                                    = 3926
	OID__INT8RANGE                                   = 3927
	OID_RECORD                                       = 2249
	OID__RECORD                                      = 2287
	OID_CSTRING                                      = 2275
	OID_ANY                                          = 2276
	OID_ANYARRAY                                     = 2277
	OID_VOID                                         = 2278
	OID_TRIGGER                                      = 2279
	OID_LANGUAGE_HANDLER                             = 2280
	OID_INTERNAL                                     = 2281
	OID_OPAQUE                                       = 2282
	OID_ANYELEMENT                                   = 2283
	OID_ANYNONARRAY                                  = 2776
	OID_ANYENUM                                      = 3500
	OID_FDW_HANDLER                                  = 3115
	OID_ANYRANGE                                     = 3831
	OID_PG_ATTRDEF                                   = 10000
	OID_PG_CONSTRAINT                                = 10001
	OID_PG_INHERITS                                  = 10002
	OID_PG_INDEX                                     = 10003
	OID_PG_OPERATOR                                  = 10004
	OID_PG_OPFAMILY                                  = 10005
	OID_PG_OPCLASS                                   = 10006
	OID_PG_AM                                        = 10116
	OID_PG_AMOP                                      = 10117
	OID_PG_AMPROC                                    = 10511
	OID_PG_LANGUAGE                                  = 10798
	OID_PG_LARGEOBJECT_METADATA                      = 10799
	OID_PG_LARGEOBJECT                               = 10800
	OID_PG_AGGREGATE                                 = 10801
	OID_PG_STATISTIC                                 = 10802
	OID_PG_REWRITE                                   = 10803
	OID_PG_TRIGGER                                   = 10804
	OID_PG_DESCRIPTION                               = 10805
	OID_PG_CAST                                      = 10806
	OID_PG_ENUM                                      = 11003
	OID_PG_NAMESPACE                                 = 11004
	OID_PG_CONVERSION                                = 11005
	OID_PG_DEPEND                                    = 11006
	OID_PG_DATABASE                                  = 1248
	OID_PG_DB_ROLE_SETTING                           = 11007
	OID_PG_TABLESPACE                                = 11008
	OID_PG_PLTEMPLATE                                = 11009
	OID_PG_AUTHID                                    = 2842
	OID_PG_AUTH_MEMBERS                              = 2843
	OID_PG_SHDEPEND                                  = 11010
	OID_PG_SHDESCRIPTION                             = 11011
	OID_PG_TS_CONFIG                                 = 11012
	OID_PG_TS_CONFIG_MAP                             = 11013
	OID_PG_TS_DICT                                   = 11014
	OID_PG_TS_PARSER                                 = 11015
	OID_PG_TS_TEMPLATE                               = 11016
	OID_PG_EXTENSION                                 = 11017
	OID_PG_FOREIGN_DATA_WRAPPER                      = 11018
	OID_PG_FOREIGN_SERVER                            = 11019
	OID_PG_USER_MAPPING                              = 11020
	OID_PG_FOREIGN_TABLE                             = 11021
	OID_PG_DEFAULT_ACL                               = 11022
	OID_PG_SECLABEL                                  = 11023
	OID_PG_SHSECLABEL                                = 11024
	OID_PG_COLLATION                                 = 11025
	OID_PG_RANGE                                     = 11026
	OID_PG_TOAST_2604                                = 11027
	OID_PG_TOAST_2606                                = 11028
	OID_PG_TOAST_2609                                = 11029
	OID_PG_TOAST_1255                                = 11030
	OID_PG_TOAST_2618                                = 11031
	OID_PG_TOAST_3596                                = 11032
	OID_PG_TOAST_2619                                = 11033
	OID_PG_TOAST_2620                                = 11034
	OID_PG_TOAST_2396                                = 11035
	OID_PG_TOAST_2964                                = 11036
	OID_PG_ROLES                                     = 11038
	OID_PG_SHADOW                                    = 11041
	OID_PG_GROUP                                     = 11044
	OID_PG_USER                                      = 11047
	OID_PG_RULES                                     = 11050
	OID_PG_VIEWS                                     = 11054
	OID_PG_TABLES                                    = 11058
	OID_PG_INDEXES                                   = 11062
	OID_PG_STATS                                     = 11066
	OID_PG_LOCKS                                     = 11070
	OID_PG_CURSORS                                   = 11073
	OID_PG_AVAILABLE_EXTENSIONS                      = 11076
	OID_PG_AVAILABLE_EXTENSION_VERSIONS              = 11079
	OID_PG_PREPARED_XACTS                            = 11082
	OID_PG_PREPARED_STATEMENTS                       = 11086
	OID_PG_SECLABELS                                 = 11089
	OID_PG_SETTINGS                                  = 11093
	OID_PG_TIMEZONE_ABBREVS                          = 11098
	OID_PG_TIMEZONE_NAMES                            = 11101
	OID_PG_STAT_ALL_TABLES                           = 11104
	OID_PG_STAT_XACT_ALL_TABLES                      = 11108
	OID_PG_STAT_SYS_TABLES                           = 11112
	OID_PG_STAT_XACT_SYS_TABLES                      = 11116
	OID_PG_STAT_USER_TABLES                          = 11119
	OID_PG_STAT_XACT_USER_TABLES                     = 11123
	OID_PG_STATIO_ALL_TABLES                         = 11126
	OID_PG_STATIO_SYS_TABLES                         = 11130
	OID_PG_STATIO_USER_TABLES                        = 11133
	OID_PG_STAT_ALL_INDEXES                          = 11136
	OID_PG_STAT_SYS_INDEXES                          = 11140
	OID_PG_STAT_USER_INDEXES                         = 11143
	OID_PG_STATIO_ALL_INDEXES                        = 11146
	OID_PG_STATIO_SYS_INDEXES                        = 11150
	OID_PG_STATIO_USER_INDEXES                       = 11153
	OID_PG_STATIO_ALL_SEQUENCES                      = 11156
	OID_PG_STATIO_SYS_SEQUENCES                      = 11160
	OID_PG_STATIO_USER_SEQUENCES                     = 11163
	OID_PG_STAT_ACTIVITY                             = 11166
	OID_PG_STAT_REPLICATION                          = 11169
	OID_PG_STAT_DATABASE                             = 11172
	OID_PG_STAT_DATABASE_CONFLICTS                   = 11175
	OID_PG_STAT_USER_FUNCTIONS                       = 11178
	OID_PG_STAT_XACT_USER_FUNCTIONS                  = 11182
	OID_PG_STAT_BGWRITER                             = 11186
	OID_PG_USER_MAPPINGS                             = 11189
	OID_CARDINAL_NUMBER                              = 11510
	OID_CHARACTER_DATA                               = 11512
	OID_SQL_IDENTIFIER                               = 11513
	OID_INFORMATION_SCHEMA_CATALOG_NAME              = 11515
	OID_TIME_STAMP                                   = 11517
	OID_YES_OR_NO                                    = 11518
	OID_APPLICABLE_ROLES                             = 11521
	OID_ADMINISTRABLE_ROLE_AUTHORIZATIONS            = 11525
	OID_ATTRIBUTES                                   = 11528
	OID_CHARACTER_SETS                               = 11532
	OID_CHECK_CONSTRAINT_ROUTINE_USAGE               = 11536
	OID_CHECK_CONSTRAINTS                            = 11540
	OID_COLLATIONS                                   = 11544
	OID_COLLATION_CHARACTER_SET_APPLICABILITY        = 11547
	OID_COLUMN_DOMAIN_USAGE                          = 11550
	OID_COLUMN_PRIVILEGES                            = 11554
	OID_COLUMN_UDT_USAGE                             = 11558
	OID_COLUMNS                                      = 11562
	OID_CONSTRAINT_COLUMN_USAGE                      = 11566
	OID_CONSTRAINT_TABLE_USAGE                       = 11570
	OID_DOMAIN_CONSTRAINTS                           = 11574
	OID_DOMAIN_UDT_USAGE                             = 11578
	OID_DOMAINS                                      = 11581
	OID_ENABLED_ROLES                                = 11585
	OID_KEY_COLUMN_USAGE                             = 11588
	OID_PARAMETERS                                   = 11592
	OID_REFERENTIAL_CONSTRAINTS                      = 11596
	OID_ROLE_COLUMN_GRANTS                           = 11600
	OID_ROUTINE_PRIVILEGES                           = 11603
	OID_ROLE_ROUTINE_GRANTS                          = 11607
	OID_ROUTINES                                     = 11610
	OID_SCHEMATA                                     = 11614
	OID_SEQUENCES                                    = 11617
	OID_SQL_FEATURES                                 = 11621
	OID_PG_TOAST_11620                               = 11623
	OID_SQL_IMPLEMENTATION_INFO                      = 11626
	OID_PG_TOAST_11625                               = 11628
	OID_SQL_LANGUAGES                                = 11631
	OID_PG_TOAST_11630                               = 11633
	OID_SQL_PACKAGES                                 = 11636
	OID_PG_TOAST_11635                               = 11638
	OID_SQL_PARTS                                    = 11641
	OID_PG_TOAST_11640                               = 11643
	OID_SQL_SIZING                                   = 11646
	OID_PG_TOAST_11645                               = 11648
	OID_SQL_SIZING_PROFILES                          = 11651
	OID_PG_TOAST_11650                               = 11653
	OID_TABLE_CONSTRAINTS                            = 11656
	OID_TABLE_PRIVILEGES                             = 11660
	OID_ROLE_TABLE_GRANTS                            = 11664
	OID_TABLES                                       = 11667
	OID_TRIGGERED_UPDATE_COLUMNS                     = 11671
	OID_TRIGGERS                                     = 11675
	OID_UDT_PRIVILEGES                               = 11679
	OID_ROLE_UDT_GRANTS                              = 11683
	OID_USAGE_PRIVILEGES                             = 11686
	OID_ROLE_USAGE_GRANTS                            = 11690
	OID_USER_DEFINED_TYPES                           = 11693
	OID_VIEW_COLUMN_USAGE                            = 11697
	OID_VIEW_ROUTINE_USAGE                           = 11701
	OID_VIEW_TABLE_USAGE                             = 11705
	OID_VIEWS                                        = 11709
	OID_DATA_TYPE_PRIVILEGES                         = 11713
	OID_ELEMENT_TYPES                                = 11717
	OID__PG_FOREIGN_TABLE_COLUMNS                    = 11721
	OID_COLUMN_OPTIONS                               = 11725
	OID__PG_FOREIGN_DATA_WRAPPERS                    = 11728
	OID_FOREIGN_DATA_WRAPPER_OPTIONS                 = 11731
	OID_FOREIGN_DATA_WRAPPERS                        = 11734
	OID__PG_FOREIGN_SERVERS                          = 11737
	OID_FOREIGN_SERVER_OPTIONS                       = 11740
	OID_FOREIGN_SERVERS                              = 11743
	OID__PG_FOREIGN_TABLES                           = 11746
	OID_FOREIGN_TABLE_OPTIONS                        = 11750
	OID_FOREIGN_TABLES                               = 11753
	OID__PG_USER_MAPPINGS                            = 11756
	OID_USER_MAPPING_OPTIONS                         = 11759
	OID_USER_MAPPINGS                                = 11763
)
