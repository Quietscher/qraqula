package app

import (
	"context"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/qraqula/qla/internal/config"
	"github.com/qraqula/qla/internal/editor"
	"github.com/qraqula/qla/internal/endpoint"
	"github.com/qraqula/qla/internal/graphql"
	"github.com/qraqula/qla/internal/history"
	"github.com/qraqula/qla/internal/overlay"
	"github.com/qraqula/qla/internal/results"
	"github.com/qraqula/qla/internal/schema"
	"github.com/qraqula/qla/internal/statusbar"
	"github.com/qraqula/qla/internal/validate"
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
	schemaAST *validate.SchemaAST
	gqlClient *graphql.Client

	histSidebar history.Sidebar
	histStore   *history.Store
	sidebarOpen bool // user preference (ctrl+b toggle)

	configStore *config.Store
	overlay     overlay.Model

	cancelQuery    context.CancelFunc
	rightPanelMode rightPanelMode

	focus        Panel
	querying     bool
	lastEndpoint string
	width        int
	height       int

	// Timer generation counters for debouncing
	statusClearGen int
	lintGen        int

	// Cached panel dimensions from layoutPanels (content size, excluding border)
	sidebarW int // 3-panel mode only
	midW     int // 3-panel mode only
	leftW    int // 2-panel mode only
	rightW   int
	editorH  int
	varsH    int
	contentH int // full-height panels (sidebar, results)
}

// shouldShowSidebar returns true when the sidebar should actually be rendered.
// Requires both user preference AND history content.
func (m Model) shouldShowSidebar() bool {
	return m.sidebarOpen && m.histStore.HasContent()
}

func NewModel() Model {
	ed := editor.New()
	ed.Focus()

	cfgDir := defaultConfigDir()
	histDir := filepath.Join(cfgDir, "history")
	store := history.NewStore(histDir)
	_ = store.Load()
	sidebarOpen := store.Meta.SidebarOpen

	cfgStore := config.NewStore(cfgDir)
	_ = cfgStore.Load()

	ep := endpoint.New()
	vars := variables.New()

	// Restore last session state
	meta := store.Meta
	if meta.LastEnvName != "" {
		// Try to find the env in current config
		cfgStore.Config.ActiveEnv = meta.LastEnvName
		if env := cfgStore.Config.ActiveEnvironment(); env != nil {
			ep.SetValue(env.Endpoint)
			ep.SetEnvName(env.Name)
		} else {
			// Env was deleted — fall back to no env
			cfgStore.Config.ActiveEnv = ""
		}
	} else if env := cfgStore.Config.ActiveEnvironment(); env != nil {
		ep.SetValue(env.Endpoint)
		ep.SetEnvName(env.Name)
	}
	if meta.LastEndpoint != "" {
		ep.SetValue(meta.LastEndpoint)
	}
	if meta.LastQuery != "" {
		ed.SetValue(meta.LastQuery)
	}
	if meta.LastVariables != "" {
		vars.SetValue(meta.LastVariables)
	}

	return Model{
		endpoint:    ep,
		editor:      ed,
		variables:   vars,
		results:     results.New(80, 20),
		statusbar:   statusbar.New(),
		browser:     schema.NewBrowser(),
		gqlClient:   graphql.NewClient(),
		histStore:   store,
		histSidebar: history.NewSidebar(store),
		sidebarOpen: sidebarOpen,
		configStore: cfgStore,
		overlay:     overlay.New(),
		focus:       PanelEditor,
	}
}

// NewModelWithStores creates a Model with custom stores (for testing).
func NewModelWithStores(histStore *history.Store, cfgStore *config.Store) Model {
	ed := editor.New()
	ed.Focus()

	ep := endpoint.New()
	if cfgStore != nil {
		if env := cfgStore.Config.ActiveEnvironment(); env != nil {
			ep.SetValue(env.Endpoint)
			ep.SetEnvName(env.Name)
		}
	} else {
		cfgStore = config.NewStore("")
	}

	return Model{
		endpoint:    ep,
		editor:      ed,
		variables:   variables.New(),
		results:     results.New(80, 20),
		statusbar:   statusbar.New(),
		browser:     schema.NewBrowser(),
		gqlClient:   graphql.NewClient(),
		histStore:   histStore,
		histSidebar: history.NewSidebar(histStore),
		sidebarOpen: histStore.Meta.SidebarOpen,
		configStore: cfgStore,
		overlay:     overlay.New(),
		focus:       PanelEditor,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.editor.Focus(), m.autoFetchSchema())
}

func defaultConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(configDir, "qraqula")
}
