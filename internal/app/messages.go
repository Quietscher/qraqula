package app

import (
	"github.com/qraqula/qla/internal/graphql"
	"github.com/qraqula/qla/internal/schema"
)

// QueryResultMsg is sent when a query completes successfully.
type QueryResultMsg struct {
	Result *graphql.Result
}

// QueryErrorMsg is sent when a query fails.
type QueryErrorMsg struct {
	Err error
}

// QueryAbortedMsg is sent when a query is cancelled.
type QueryAbortedMsg struct{}

// SchemaFetchedMsg is sent when schema introspection completes.
type SchemaFetchedMsg struct {
	Schema *schema.Schema
}

// SchemaFetchErrorMsg is sent when schema introspection fails.
type SchemaFetchErrorMsg struct {
	Err error
}

// EditorFinishedMsg is sent when the external editor process completes.
type EditorFinishedMsg struct {
	Content string
	Panel   Panel
	Err     error
}

// statusClearMsg fires after a delay to clear the status bar error.
type statusClearMsg struct{ gen int }

// lintMsg fires after a debounce delay to lint the editor content.
type lintMsg struct{ gen int }
