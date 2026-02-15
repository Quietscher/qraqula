package app

import "github.com/qraqula/qla/internal/graphql"

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
