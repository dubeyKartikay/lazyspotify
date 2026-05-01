package player

import "testing"

func TestNewButtonsOrder(t *testing.T) {
	buttons := newButtons()

	got := make([]ButtonKind, 0, len(buttons))
	for _, button := range buttons {
		got = append(got, button.kind)
	}

	want := []ButtonKind{
		PreviousButton,
		SeekBackwardButton,
		PlayButton,
		PauseButton,
		SeekForwardButton,
		NextButton,
	}

	if len(got) != len(want) {
		t.Fatalf("button count = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("button[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestHandleButtonPressFindsButtonByKind(t *testing.T) {
	model := NewModel()

	model.HandleButtonPress(NextButton)

	for _, button := range model.controls {
		if button.kind == NextButton && !button.pressed {
			t.Fatal("expected next button to be pressed")
		}
		if button.kind != NextButton && button.pressed {
			t.Fatalf("button %v unexpectedly pressed", button.kind)
		}
	}
}
