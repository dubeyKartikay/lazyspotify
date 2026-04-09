package v1

import (
	"charm.land/lipgloss/v2"
)

type Spoke struct {
	Frames       []Frame
	currentFrame int
}

type Frame struct {
	view string
}

func NewFrame(view string) Frame {
	return Frame{
		view: view,
	}
}

func setFrames() []Frame {
	top := " .---. "
	bottom := " '---' "

	frame1 := top + "\n" + "/  |  \\" + "\n" + "|  o  |" + "\n" + "\\  |  /" + "\n" + bottom
	frame2 := top + "\n" + "/   / \\" + "\n" + "|  o  |" + "\n" + "\\ /   /" + "\n" + bottom
	frame3 := top + "\n" + "/     \\" + "\n" + "|--o--|" + "\n" + "\\     /" + "\n" + bottom
	frame4 := top + "\n" + "/ \\   \\" + "\n" + "|  o  |" + "\n" + "\\   \\ /" + "\n" + bottom

	frames := []Frame{
		NewFrame(frame1),
		NewFrame(frame2),
		NewFrame(frame3),
		NewFrame(frame4),
	}
	return frames
}

func NewSpoke() Spoke {
	return Spoke{
		Frames: setFrames(),
	}
}

func (s *Spoke) GetSize() (int, int) {
	return lipgloss.Width(s.Frames[s.currentFrame].view),
		lipgloss.Height(s.Frames[s.currentFrame].view)
}

func (s *Spoke) View() string {
	return s.Frames[s.currentFrame].view
}

func (s *Spoke) NextFrame() {
	s.currentFrame = (s.currentFrame + 1) % len(s.Frames)
}

