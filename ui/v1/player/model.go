package player

import (
	"charm.land/lipgloss/v2"
)

type ButtonKind int

const (
	PlayButton ButtonKind = iota
	PauseButton
	SeekForwardButton
	SeekBackwardButton
	NextButton
	PreviousButton
)

type Status struct {
	PlayerReady bool
	Playing     bool
	Position    int
	Duration    int
	Volume      int
	MaxVolume   int
}

type Model struct {
	style    lipgloss.Style
	cassette cassette
	controls []button
}

type button struct {
	kind    ButtonKind
	style   lipgloss.Style
	icon    string
	pressed bool
}

type buttonSpec struct {
	kind ButtonKind
	icon string
}

func NewModel() Model {
	return Model{
		style:    lipgloss.NewStyle(),
		cassette: newCassette(),
		controls: newButtons(),
	}
}

func newButtons() []button {
	specs := []buttonSpec{
		{kind: PreviousButton, icon: "|<"},
		{kind: SeekBackwardButton, icon: "<<"},
		{kind: PlayButton, icon: "|>"},
		{kind: PauseButton, icon: "||"},
		{kind: SeekForwardButton, icon: ">>"},
		{kind: NextButton, icon: ">|"},
	}
	return buildButtons(specs)
}

func buildButtons(specs []buttonSpec) []button {
	return commonButtons(specs, func(spec buttonSpec) button {
		return button{
			kind:  spec.kind,
			style: lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
			icon:  spec.icon,
		}
	})
}

func commonButtons[T any, U any](items []T, mapFn func(T) U) []U {
	mapped := make([]U, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, mapFn(item))
	}
	return mapped
}
