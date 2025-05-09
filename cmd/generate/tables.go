// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "log"

// prop combines the name of a property with the name of the structure it is in.
type prop [2]string

const (
	nothing = iota
	wantStar
	wantOpt
	wantOptStar
)

// goplsStar records the optionality of each field in the protocol.
// The comments are vague hints as to why removing the line is not trivial.
// A.B.C.D means that one of B or C would change to a pointer
// so a test or initialization would be needed
var goplsStar = map[prop]int{
	{"AnnotatedTextEdit", "annotationId"}:  wantOptStar,
	{"ClientCapabilities", "textDocument"}: wantOpt, // A.B.C.D at fake/editor.go:255
	{"ClientCapabilities", "window"}:       wantOpt, // test failures
	{"ClientCapabilities", "workspace"}:    wantOpt, // test failures
	{"CodeAction", "kind"}:                 wantOpt, // A.B.C.D

	{"CodeActionClientCapabilities", "codeActionLiteralSupport"}: wantOpt, // test failures

	{"CompletionClientCapabilities", "completionItem"}: wantOpt, // A.B.C.D
	{"CompletionClientCapabilities", "insertTextMode"}: wantOpt, // A.B.C.D
	{"CompletionItem", "kind"}:                         wantOpt, // need temporary variables
	{"CompletionParams", "context"}:                    wantOpt, // needs nil checks

	{"Diagnostic", "severity"}:            wantOpt,     // nil checks or more careful thought
	{"DidSaveTextDocumentParams", "text"}: wantOptStar, // capabilities_test.go:112 logic
	{"DocumentHighlight", "kind"}:         wantOpt,     // need temporary variables
	{"Hover", "range"}:                    wantOpt,     // complex expressions
	{"InlayHint", "kind"}:                 wantOpt,     // temporary variables

	{"TextDocumentClientCapabilities", "codeAction"}:          wantOpt,     // A.B.C.D
	{"TextDocumentClientCapabilities", "completion"}:          wantOpt,     // A.B.C.D
	{"TextDocumentClientCapabilities", "documentSymbol"}:      wantOpt,     // A.B.C.D
	{"TextDocumentClientCapabilities", "publishDiagnostics"}:  wantOpt,     //A.B.C.D
	{"TextDocumentClientCapabilities", "semanticTokens"}:      wantOpt,     // A.B.C.D
	{"TextDocumentContentChangePartial", "range"}:             wantOptStar, // == nil test
	{"TextDocumentSyncOptions", "change"}:                     wantOpt,     // &constant
	{"WorkDoneProgressParams", "workDoneToken"}:               wantOpt,     // test failures
	{"WorkspaceClientCapabilities", "didChangeConfiguration"}: wantOpt,     // A.B.C.D
	{"WorkspaceClientCapabilities", "didChangeWatchedFiles"}:  wantOpt,     // A.B.C.D
}

// keep track of which entries in goplsStar are used
var usedGoplsStar = make(map[prop]bool)

// For gopls compatibility, use a different, typically more restrictive, type for some fields.
var renameProp = map[prop]string{
	{"CancelParams", "id"}:   "interface{}",
	{"Command", "arguments"}: "[]json.RawMessage",
	{"CodeAction", "data"}:   "json.RawMessage", // delay unmarshalling commands
	{"Diagnostic", "code"}:   "interface{}",
	{"Diagnostic", "data"}:   "json.RawMessage", // delay unmarshalling quickfixes

	{"DocumentDiagnosticReportPartialResult", "relatedDocuments"}: "map[DocumentUri]interface{}",

	{"ExecuteCommandParams", "arguments"}: "[]json.RawMessage",
	{"FoldingRange", "kind"}:              "string",
	{"Hover", "contents"}:                 "MarkupContent",
	{"InlayHint", "label"}:                "[]InlayHintLabelPart",

	{"RelatedFullDocumentDiagnosticReport", "relatedDocuments"}:      "map[DocumentUri]interface{}",
	{"RelatedUnchangedDocumentDiagnosticReport", "relatedDocuments"}: "map[DocumentUri]interface{}",

	// PJW: this one is tricky.
	{"ServerCapabilities", "codeActionProvider"}: "interface{}",

	{"ServerCapabilities", "inlayHintProvider"}: "interface{}",
	// slightly tricky
	{"ServerCapabilities", "renameProvider"}: "interface{}",
	// slightly tricky
	{"ServerCapabilities", "semanticTokensProvider"}: "interface{}",
	// slightly tricky
	{"ServerCapabilities", "textDocumentSync"}: "interface{}",
	{"TextDocumentSyncOptions", "save"}:        "SaveOptions",
	{"WorkspaceEdit", "documentChanges"}:       "[]DocumentChange",
}

// which entries of renameProp were used
var usedRenameProp = make(map[prop]bool)

type adjust struct {
	prefix, suffix string
}

// disambiguate specifies prefixes or suffixes to add to all values of
// some enum types to avoid name conflicts
var disambiguate = map[string]adjust{
	"CodeActionTriggerKind":        {"CodeAction", ""},
	"CompletionItemKind":           {"", "Completion"},
	"CompletionItemTag":            {"Compl", ""},
	"DiagnosticSeverity":           {"Severity", ""},
	"DocumentDiagnosticReportKind": {"Diagnostic", ""},
	"FileOperationPatternKind":     {"", "Pattern"},
	"InlineCompletionTriggerKind":  {"Inline", ""},
	"InsertTextFormat":             {"", "TextFormat"},
	"LanguageKind":                 {"Lang", ""},
	"SemanticTokenModifiers":       {"Mod", ""},
	"SemanticTokenTypes":           {"", "Type"},
	"SignatureHelpTriggerKind":     {"Sig", ""},
	"SymbolTag":                    {"", "Symbol"},
	"WatchKind":                    {"Watch", ""},
}

// which entries of disambiguate got used
var usedDisambiguate = make(map[string]bool)

// additionalFields defines extra fields to add to specific types during generation
// The key is the Go struct name (after applying goName), the value is a slice of field definitions
var additionalFields = map[string][]string{
	"BaseSymbolInformation": {
		"Score float64 `json:\"score,omitempty\"` // added for clangd compatibility",
	},
}

// For spec compliance, we keep only essential type mappings that don't override OR types
var goplsType = map[string]string{
	"ConfigurationParams": "ParamConfiguration",
	// "DocumentUri":         "DocumentUri",
	"InitializeParams": "ParamInitialize",
	"LSPAny":           "any",
	"[]LSPAny":         "[]any",
	"[]uinteger":       "[]uint32",
	"boolean":          "bool",
	"decimal":          "float64",
	"integer":          "int32",
	"uinteger":         "uint32",
}

var usedGoplsType = make(map[string]bool)

// methodNames is a map from the method to the name of the function that handles it
var methodNames = map[string]string{
	"$/cancelRequest":                        "CancelRequest",
	"$/logTrace":                             "LogTrace",
	"$/progress":                             "Progress",
	"$/setTrace":                             "SetTrace",
	"callHierarchy/incomingCalls":            "IncomingCalls",
	"callHierarchy/outgoingCalls":            "OutgoingCalls",
	"client/registerCapability":              "RegisterCapability",
	"client/unregisterCapability":            "UnregisterCapability",
	"codeAction/resolve":                     "ResolveCodeAction",
	"codeLens/resolve":                       "ResolveCodeLens",
	"completionItem/resolve":                 "ResolveCompletionItem",
	"documentLink/resolve":                   "ResolveDocumentLink",
	"exit":                                   "Exit",
	"initialize":                             "Initialize",
	"initialized":                            "Initialized",
	"inlayHint/resolve":                      "Resolve",
	"notebookDocument/didChange":             "DidChangeNotebookDocument",
	"notebookDocument/didClose":              "DidCloseNotebookDocument",
	"notebookDocument/didOpen":               "DidOpenNotebookDocument",
	"notebookDocument/didSave":               "DidSaveNotebookDocument",
	"shutdown":                               "Shutdown",
	"telemetry/event":                        "Event",
	"textDocument/codeAction":                "CodeAction",
	"textDocument/codeLens":                  "CodeLens",
	"textDocument/colorPresentation":         "ColorPresentation",
	"textDocument/completion":                "Completion",
	"textDocument/declaration":               "Declaration",
	"textDocument/definition":                "Definition",
	"textDocument/diagnostic":                "Diagnostic",
	"textDocument/didChange":                 "DidChange",
	"textDocument/didClose":                  "DidClose",
	"textDocument/didOpen":                   "DidOpen",
	"textDocument/didSave":                   "DidSave",
	"textDocument/documentColor":             "DocumentColor",
	"textDocument/documentHighlight":         "DocumentHighlight",
	"textDocument/documentLink":              "DocumentLink",
	"textDocument/documentSymbol":            "DocumentSymbol",
	"textDocument/foldingRange":              "FoldingRange",
	"textDocument/formatting":                "Formatting",
	"textDocument/hover":                     "Hover",
	"textDocument/implementation":            "Implementation",
	"textDocument/inlayHint":                 "InlayHint",
	"textDocument/inlineCompletion":          "InlineCompletion",
	"textDocument/inlineValue":               "InlineValue",
	"textDocument/linkedEditingRange":        "LinkedEditingRange",
	"textDocument/moniker":                   "Moniker",
	"textDocument/onTypeFormatting":          "OnTypeFormatting",
	"textDocument/prepareCallHierarchy":      "PrepareCallHierarchy",
	"textDocument/prepareRename":             "PrepareRename",
	"textDocument/prepareTypeHierarchy":      "PrepareTypeHierarchy",
	"textDocument/publishDiagnostics":        "PublishDiagnostics",
	"textDocument/rangeFormatting":           "RangeFormatting",
	"textDocument/rangesFormatting":          "RangesFormatting",
	"textDocument/references":                "References",
	"textDocument/rename":                    "Rename",
	"textDocument/selectionRange":            "SelectionRange",
	"textDocument/semanticTokens/full":       "SemanticTokensFull",
	"textDocument/semanticTokens/full/delta": "SemanticTokensFullDelta",
	"textDocument/semanticTokens/range":      "SemanticTokensRange",
	"textDocument/signatureHelp":             "SignatureHelp",
	"textDocument/typeDefinition":            "TypeDefinition",
	"textDocument/willSave":                  "WillSave",
	"textDocument/willSaveWaitUntil":         "WillSaveWaitUntil",
	"typeHierarchy/subtypes":                 "Subtypes",
	"typeHierarchy/supertypes":               "Supertypes",
	"window/logMessage":                      "LogMessage",
	"window/showDocument":                    "ShowDocument",
	"window/showMessage":                     "ShowMessage",
	"window/showMessageRequest":              "ShowMessageRequest",
	"window/workDoneProgress/cancel":         "WorkDoneProgressCancel",
	"window/workDoneProgress/create":         "WorkDoneProgressCreate",
	"workspace/applyEdit":                    "ApplyEdit",
	"workspace/codeLens/refresh":             "CodeLensRefresh",
	"workspace/configuration":                "Configuration",
	"workspace/diagnostic":                   "DiagnosticWorkspace",
	"workspace/diagnostic/refresh":           "DiagnosticRefresh",
	"workspace/didChangeConfiguration":       "DidChangeConfiguration",
	"workspace/didChangeWatchedFiles":        "DidChangeWatchedFiles",
	"workspace/didChangeWorkspaceFolders":    "DidChangeWorkspaceFolders",
	"workspace/didCreateFiles":               "DidCreateFiles",
	"workspace/didDeleteFiles":               "DidDeleteFiles",
	"workspace/didRenameFiles":               "DidRenameFiles",
	"workspace/executeCommand":               "ExecuteCommand",
	"workspace/foldingRange/refresh":         "FoldingRangeRefresh",
	"workspace/inlayHint/refresh":            "InlayHintRefresh",
	"workspace/inlineValue/refresh":          "InlineValueRefresh",
	"workspace/semanticTokens/refresh":       "SemanticTokensRefresh",
	"workspace/symbol":                       "Symbol",
	"workspace/textDocumentContent":          "TextDocumentContent",
	"workspace/textDocumentContent/refresh":  "TextDocumentContentRefresh",
	"workspace/willCreateFiles":              "WillCreateFiles",
	"workspace/willDeleteFiles":              "WillDeleteFiles",
	"workspace/willRenameFiles":              "WillRenameFiles",
	"workspace/workspaceFolders":             "WorkspaceFolders",
	"workspaceSymbol/resolve":                "ResolveWorkspaceSymbol",
}

func methodName(method string) string {
	ans := methodNames[method]
	if ans == "" {
		log.Fatalf("unknown method %q", method)
	}
	return ans
}
