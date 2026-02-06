package mcpserver

// ViewDisassemblyArgs are the arguments for the view_disassembly tool.
type ViewDisassemblyArgs struct {
	StartAddr string  `json:"start_addr" jsonschema:"Start address in hex (e.g. '$C000' or '0xC000')"`
	EndAddr   *string `json:"end_addr,omitempty" jsonschema:"End address in hex (inclusive). If not provided, shows ~32 bytes from start."`
}

// AddAnnotationArgs are the arguments for the add_annotation tool.
type AddAnnotationArgs struct {
	Address string `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000')"`
	Comment string `json:"comment" jsonschema:"The annotation text explaining the instruction's purpose"`
}

// AddHeadlineArgs are the arguments for the add_headline tool.
type AddHeadlineArgs struct {
	Address string `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000') - typically a subroutine entry point"`
	Comment string `json:"comment" jsonschema:"The headline text describing the subroutine or section (can be multi-line)"`
}

// AddSymbolArgs are the arguments for the add_symbol tool.
type AddSymbolArgs struct {
	Address    string  `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000')"`
	Name       string  `json:"name" jsonschema:"Symbol name (e.g. 'init_screen', 'sprite_x', 'VIC_CTRL1')"`
	SymbolType *string `json:"symbol_type,omitempty" jsonschema:"Type of symbol,enum=subroutine,enum=label,enum=byte,enum=word"`
}

// QuerySymbolsArgs are the arguments for the query_symbols tool.
type QuerySymbolsArgs struct {
	StartAddr  *string `json:"start_addr,omitempty" jsonschema:"Start address for range filter (optional)"`
	EndAddr    *string `json:"end_addr,omitempty" jsonschema:"End address for range filter (optional)"`
	SymbolType *string `json:"symbol_type,omitempty" jsonschema:"Filter by symbol type (optional),enum=subroutine,enum=label,enum=byte,enum=word,enum=entry"`
	NameFilter *string `json:"name_filter,omitempty" jsonschema:"Filter symbols containing this substring (case-insensitive, optional)"`
}

// QueryXRefsArgs are the arguments for the query_xrefs tool.
type QueryXRefsArgs struct {
	Address   string  `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000')"`
	Direction *string `json:"direction,omitempty" jsonschema:"Direction of references: 'to' (who references this address), 'from' (what this address references), or 'both',enum=to,enum=from,enum=both"`
}

// ListSubroutinesArgs are the arguments for the list_subroutines tool.
type ListSubroutinesArgs struct {
	StartAddr *string `json:"start_addr,omitempty" jsonschema:"Start address for range filter (optional, defaults to binary start)"`
	EndAddr   *string `json:"end_addr,omitempty" jsonschema:"End address for range filter (optional, defaults to binary end)"`
}

// GetSubroutineContextArgs are the arguments for the get_subroutine_context tool.
type GetSubroutineContextArgs struct {
	Address string `json:"address" jsonschema:"Subroutine address in hex (e.g. '$C000' or '0xC000')"`
}
