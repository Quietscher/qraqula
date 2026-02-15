package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qraqula/qla/internal/editor"
	"github.com/qraqula/qla/internal/endpoint"
	"github.com/qraqula/qla/internal/graphql"
	"github.com/qraqula/qla/internal/results"
	"github.com/qraqula/qla/internal/statusbar"
	"github.com/qraqula/qla/internal/variables"
)

type Model struct {
	endpoint  endpoint.Model
	editor    editor.Model
	variables variables.Model
	results   results.Model
	statusbar statusbar.Model

	gqlClient   *graphql.Client
	cancelQuery context.CancelFunc

	focus    Panel
	querying bool
	width    int
	height   int
}

func NewModel() Model {
	ed := editor.New()
	ed.Focus()

	return Model{
		endpoint:  endpoint.New(),
		editor:    ed,
		variables: variables.New(),
		results:   results.New(80, 20),
		statusbar: statusbar.New(),
		gqlClient: graphql.NewClient(),
		focus:     PanelEditor,
	}
}

func (m Model) Init() tea.Cmd {
	return m.editor.Focus()
}
