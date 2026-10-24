package sem

import (
	"strings"
	"unicode"
)

// toolHandlerKinds are the symbol kinds that can implement an agent tool: code
// that runs when the tool is called. A field, a variable or a Markdown section
// can mention a tool handler, but it cannot be one.
var toolHandlerKinds = map[string]bool{
	"function": true,
	"method":   true,
	"class":    true,
	"struct":   true,
	"type":     true,
}

// toolHandlerActions are the words that, next to "tool", name the act of
// handling a tool call. "call" is deliberately absent: agent-wrapper code is
// full of tool-call DATA (ToolCall, toolCallID, parseToolCall), and none of it
// handles anything. "schema" is absent for the same reason: a schema declares a
// tool, it does not run one.
var toolHandlerActions = map[string]bool{
	"handle":     true,
	"handler":    true,
	"handlers":   true,
	"execute":    true,
	"executor":   true,
	"invoke":     true,
	"dispatch":   true,
	"dispatcher": true,
}

// toolHandlerPredicates are leading words that make a name a question about its
// subject rather than the subject itself: looksLikeToolHandler and
// isToolHandler decide whether something handles a tool; they handle none.
var toolHandlerPredicates = map[string]bool{
	"is":     true,
	"has":    true,
	"looks":  true,
	"should": true,
	"can":    true,
	"must":   true,
	"needs":  true,
}

// mcpToolCallRequestType is the request type an MCP SDK hands to a tool
// handler (mark3labs/mcp-go and the official Go and TypeScript SDKs all spell it
// this way). A function that takes one is a registered tool handler whatever it
// is named, which is the usual shape: `handleSearch(ctx, req mcp.CallToolRequest)`.
const mcpToolCallRequestType = "CallToolRequest"

// looksLikeToolHandler reports whether a symbol handles an agent tool call, so
// that syntheticBoundarySymbols surfaces it as a kind=tool boundary.
//
// The decision reads only the symbol's identifiers: its name, its qualified
// name and its signature. It used to tokenize the whole body, comments and
// string literals included, so any function whose prose or vocabulary put
// "tool" beside "schema", "handler" or "execute" (a stop-word list, a test of
// MCP schemas, a struct whose field comments mention a tool's handler, a main
// whose comments say "tool" and which calls cli.Execute) became a
// second, synthetic definition of its own name, and every def, neighbors and
// impact lookup of that name came back ambiguous. A body says what a function
// talks about; only its name and signature say what it is.
func looksLikeToolHandler(symbol SymbolRecord) bool {
	if !toolHandlerKinds[symbol.Kind] || isTestName(symbol.Name) {
		return false
	}
	own := identifierWords(symbol.Name)
	if len(own) == 0 || toolHandlerPredicates[own[0]] {
		return false
	}
	if signatureNamesIdentifier(symbol.Signature, mcpToolCallRequestType) {
		return true
	}
	// "tool" may come from the container (SearchTool.execute), but the action
	// must be the symbol's own: ToolHandler.close is a method of a handler, not
	// a handler.
	namesTool := false
	for _, word := range identifierWords(symbol.QualifiedName) {
		if word == "tool" || word == "tools" {
			namesTool = true
			break
		}
	}
	if !namesTool {
		return false
	}
	for _, word := range own {
		if toolHandlerActions[word] {
			return true
		}
	}
	return false
}

// signatureNamesIdentifier reports whether identifier appears in signature as a
// whole identifier (mcp.CallToolRequest, *mcp.CallToolRequest), not as a
// substring of a longer one.
func signatureNamesIdentifier(signature, identifier string) bool {
	for _, field := range strings.FieldsFunc(signature, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	}) {
		if field == identifier {
			return true
		}
	}
	return false
}

// identifierWords splits an identifier into lowercase words on separators and
// on case boundaries, keeping acronyms whole: handleMCPToolCall yields
// handle/mcp/tool/call and AuthService.execute_tool_handler yields
// auth/service/execute/tool/handler.
func identifierWords(identifier string) []string {
	runes := []rune(identifier)
	var words []string
	start := -1
	flush := func(end int) {
		if start >= 0 && end > start {
			words = append(words, strings.ToLower(string(runes[start:end])))
		}
		start = -1
	}
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush(i)
			continue
		}
		if start < 0 {
			start = i
			continue
		}
		previous := runes[i-1]
		lowerToUpper := unicode.IsUpper(r) && (unicode.IsLower(previous) || unicode.IsDigit(previous))
		acronymEnd := unicode.IsUpper(r) && unicode.IsUpper(previous) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
		if lowerToUpper || acronymEnd {
			flush(i)
			start = i
		}
	}
	flush(len(runes))
	return words
}
