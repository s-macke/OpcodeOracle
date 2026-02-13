package mcpserver

// ViewDisassemblyArgs are the arguments for the view_disassembly tool.
type ViewDisassemblyArgs struct {
	StartAddr string  `json:"start_addr" jsonschema:"Start address in hex (e.g. '$C000' or '0xC000')"`
	EndAddr   *string `json:"end_addr,omitempty" jsonschema:"End address in hex (inclusive). If not provided, shows ~32 bytes from start."`
}

// SearchDisassemblyArgs are the arguments for the search_disassembly tool.
type SearchDisassemblyArgs struct {
	Query         string `json:"query" jsonschema:"Text to search for in rendered disassembly output"`
	CaseSensitive *bool  `json:"case_sensitive,omitempty" jsonschema:"Case-sensitive matching (optional, default false)"`
	ContextLines  *int   `json:"context_lines,omitempty" jsonschema:"Number of context lines before/after each match (optional, default 1, max 3)"`
	MaxResults    *int   `json:"max_results,omitempty" jsonschema:"Maximum number of matches to return (optional, default 20, max 200)"`
}

// AddAnnotationArgs are the arguments for the add_annotation tool.
type AddAnnotationArgs struct {
	Address string `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000')"`
	Comment string `json:"comment" jsonschema:"The annotation text explaining the instruction's purpose"`
	Extend  *bool  `json:"extend,omitempty" jsonschema:"Append to existing assistant annotation instead of replacing (optional, default false)"`
}

// RemoveAnnotationArgs are the arguments for the remove_annotation tool.
type RemoveAnnotationArgs struct {
	Address string  `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000')"`
	Author  *string `json:"author,omitempty" jsonschema:"Annotation author to remove (optional),enum=assistant,enum=user"`
}

// AddHeadlineArgs are the arguments for the add_headline tool.
type AddHeadlineArgs struct {
	Address string `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000') - typically a subroutine entry point"`
	Comment string `json:"comment" jsonschema:"The headline text describing the subroutine or section (can be multi-line)"`
	Extend  *bool  `json:"extend,omitempty" jsonschema:"Append to existing assistant headline instead of replacing (optional, default false)"`
}

// RemoveHeadlineArgs are the arguments for the remove_headline tool.
type RemoveHeadlineArgs struct {
	Address string  `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000')"`
	Author  *string `json:"author,omitempty" jsonschema:"Headline author to remove (optional),enum=assistant,enum=user"`
}

// AddSymbolArgs are the arguments for the add_symbol tool.
type AddSymbolArgs struct {
	Address    string  `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000')"`
	Name       string  `json:"name" jsonschema:"Symbol name (e.g. 'init_screen', 'sprite_x', 'VIC_CTRL1')"`
	SymbolType *string `json:"symbol_type,omitempty" jsonschema:"Type of symbol,enum=subroutine,enum=label,enum=byte,enum=word"`
}

// RemoveSymbolArgs are the arguments for the remove_symbol tool.
type RemoveSymbolArgs struct {
	Address string  `json:"address" jsonschema:"Address in hex (e.g. '$C000' or '0xC000')"`
	Name    *string `json:"name,omitempty" jsonschema:"Optional symbol name guard. If set, removal only happens when names match."`
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

// ReinterpretAsCodeArgs are the arguments for the reinterpret_as_code tool.
type ReinterpretAsCodeArgs struct {
	CodeAddress string `json:"code_address" jsonschema:"Single address to force as code seed"`
}

// ReinterpretAsDataArgs are the arguments for the reinterpret_as_data tool.
type ReinterpretAsDataArgs struct {
	StartAddr string `json:"start_addr" jsonschema:"Range start to force as hard-locked data"`
	EndAddr   string `json:"end_addr" jsonschema:"Range end to force as hard-locked data"`
}

// ListSubroutinesAndDataSegmentsArgs are the arguments for the list_subroutines_and_data_segments tool.
type ListSubroutinesAndDataSegmentsArgs struct {
	StartAddr *string `json:"start_addr,omitempty" jsonschema:"Start address for range filter (optional, defaults to binary start)"`
	EndAddr   *string `json:"end_addr,omitempty" jsonschema:"End address for range filter (optional, defaults to binary end)"`
}

// GetSubroutineContextArgs are the arguments for the get_subroutine_context tool.
type GetSubroutineContextArgs struct {
	Address string `json:"address" jsonschema:"Subroutine address in hex (e.g. '$C000' or '0xC000')"`
}
