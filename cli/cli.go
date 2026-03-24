package cli

import (
	"fmt"
)

func Run(args []string) {
	switch args[0] {
	case "auth":
		authHandler(args)
	case "play":
		playHandler(args)
  default:
    printUsage()
	}
}

func authHandler(args []string) {
	if len(args) > 1 {
		printUsage()
		return
	}

}

func playHandler(args []string) {
	if len(args) != 2{
		printUsage()
		return
	}
}

func printUsage() {
	fmt.Println("Usage: lazyspotify <command>")
	fmt.Println("Commands:")
	fmt.Println("  auth    Authenticate with Spotify")
	fmt.Println("  play    Play a hardcoded Spotify track")
}
