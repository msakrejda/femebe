package dogconf

// Toplevel production of the AST
type RequestSyntax struct {
	Spec SpecSyntax

	// Action like "get", "delete", et al
	Action ActionSyntax
}

// Unifies the types for all forms of specifying the target of an
// action: "all", one specific route, both with and without an OCN.
type SpecSyntax interface{}

// Prouced when a request targets all entries. 
type TargetAllSpecSyntax struct {
	// The 'all' token is retained for positioning information in
	// error reporting.
	Target *Token
}

// Produced when a command targets one entry, but not at any specific
// version (e.g. for "get").
type TargetOneSpecSyntax struct {
	What *Token
}

// Produced when a command targets one entry at a specific version
// (e.g. "patch").
type TargetOcnSpecSyntax struct {
	TargetOneSpecSyntax
	Ocn *Token
}

// Unifies the types for all "actions", e.g. get, create, patch,
// delete
type ActionSyntax interface{}

type PatchActionSyntax struct {
	// Properties to be used to update a route's record.
	PatchProps map[*Token]*Token
}

type CreateActionSyntax struct {
	// Properties to be used to create a new route record.
	CreateProps map[*Token]*Token
}

type GetActionSyntax struct {
	// To hold token information for error reporting.
	GetToken *Token
}

type DeleteActionSyntax struct {
	// To hold token information for error reporting.
	DeleteToken *Token
}
