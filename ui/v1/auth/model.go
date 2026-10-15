package auth

import (
	"charm.land/bubbles/v2/textinput"

	"github.com/dubeyKartikay/lazyspotify/core/auth"
)

type State int

const (
	NeedsAuth State = iota
	Authenticating
	Authenticated
)

type Model struct {
	authState       State
	auth            *auth.Authenticator
	authFlowUpdates chan string
	err             error
	width           int
	height          int
	copied          bool
	pasteInput      textinput.Model
	pasteFocused    bool
	pasteErr        error
	pasteSubmitting bool
}

func NewModel() *Model {
	return &Model{
		authState:       Authenticated,
		auth:            auth.New(),
		authFlowUpdates: make(chan string),
		pasteInput:      newPasteInput(),
	}
}

func newPasteInput() textinput.Model {
	input := textinput.New()
	input.Prompt = "Callback: "
	input.Placeholder = "paste the redirected browser url here"
	input.CharLimit = 2048
	return input
}

func (m *Model) State() State {
	return m.authState
}

func (m *Model) EnteringCallback() bool {
	return m.pasteFocused
}

func (m *Model) SetState(state State) {
	m.authState = state
}

func (m *Model) Authenticator() *auth.Authenticator {
	return m.auth
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.pasteInput.SetWidth(max(0, width-14))
}
