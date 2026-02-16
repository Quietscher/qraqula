package app

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/editor"
	"github.com/qraqula/qla/internal/endpoint"
	"github.com/qraqula/qla/internal/graphql"
	"github.com/qraqula/qla/internal/results"
	"github.com/qraqula/qla/internal/schema"
	"github.com/qraqula/qla/internal/statusbar"
	"github.com/qraqula/qla/internal/variables"
)

type rightPanelMode int

const (
	modeResults rightPanelMode = iota
	modeSchema
)

type Model struct {
	endpoint  endpoint.Model
	editor    editor.Model
	variables variables.Model
	results   results.Model
	statusbar statusbar.Model

	browser   schema.Browser
	gqlClient *graphql.Client

	cancelQuery    context.CancelFunc
	rightPanelMode rightPanelMode

	focus        Panel
	querying     bool
	lastEndpoint string
	width        int
	height       int
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
		browser:   schema.NewBrowser(),
		gqlClient: graphql.NewClient(),
		focus:     PanelEditor,
	}
}

func (m Model) Init() tea.Cmd {
	return m.editor.Focus()
}
