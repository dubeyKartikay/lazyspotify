package common

import (
	"testing"

	"charm.land/bubbles/v2/key"
)

func TestFullHelpIncludesMoreInfoWhenPanelOpen(t *testing.T) {
	help := NewAppKeyMap().WithMediaPanelOpen(true).FullHelp()
	if !hasHelpEntry(help, "i", "more info") {
		t.Fatal("expected full help to include i more info when panel is open")
	}
}

func TestFullHelpSwitchesToInfoModeHelp(t *testing.T) {
	help := NewAppKeyMap().WithMediaPanelOpen(true).WithInfoOpen(true).FullHelp()
	if !hasHelpEntry(help, "i/esc/del", "close info") {
		t.Fatal("expected full help to include close info entry")
	}
	if !hasHelpEntry(help, "ctrl+u/d", "scroll info") {
		t.Fatal("expected full help to include scroll info entry")
	}
	if !hasHelpEntry(help, "right/l/]", "next page") {
		t.Fatal("expected full help to include next page with brackets")
	}
	if !hasHelpEntry(help, "left/h/[", "prev page") {
		t.Fatal("expected full help to include prev page with brackets")
	}
}

func hasHelpEntry(rows [][]key.Binding, keyLabel, desc string) bool {
	for _, row := range rows {
		for _, binding := range row {
			help := binding.Help()
			if help.Key == keyLabel && help.Desc == desc {
				return true
			}
		}
	}
	return false
}
