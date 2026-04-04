package v1

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Spoke struct {
	Frames []Frame
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

type NextFrameMsg struct {}

func setFrames() []Frame{
	topStateMid := "┌──╤──┐"
	topStateEnd := "┌─────┐"
	midLayer1Mid := "│  |  │"
	midLayer1End := "│ \\ / │"
	midLayer2Mid := "│──┼──│"
	midLayer2End := "│  X  │"
	midLayer3Mid := "│  |  │"
	midLayer3End := "│ / \\ │"
	bottomStateMid := "└──╧──┘"
	bottomStateEnd := "└─────┘"

	frame1 := topStateMid + "\n" + midLayer1Mid + "\n" + midLayer2Mid + "\n" + midLayer3Mid + "\n" + bottomStateMid
	frame2 := topStateEnd + "\n" + midLayer1End + "\n" + midLayer2End + "\n" + midLayer3End + "\n" + bottomStateEnd

	frames := []Frame{
		NewFrame(frame1),
		NewFrame(frame2),
	}
	return frames
}


func NewSpoke(width, height int) Spoke {
	return Spoke{
		Frames: setFrames(),
	}
}

func (s *Spoke) GetSize () (int, int) {
	return lipgloss.Width(s.Frames[s.currentFrame].view), 
	lipgloss.Height(s.Frames[s.currentFrame].view)
}

func (s *Spoke) View() string {
	return s.Frames[s.currentFrame].view
}

func (s *Spoke) NextFrame() {
	s.currentFrame = (s.currentFrame + 1) % len(s.Frames)
}

func DoTickSpokes() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return NextFrameMsg{}
	})
}
