package main
import ("flag"
				"fmt"
				"github.com/dubeyKartikay/lazyspotify/cli"
)
func main() {
	flag.Parse()
	switch {
		case flag.NArg() > 0:
      cli.Authtenticate()
    default:
      fmt.Println("Running the TUI")
	}
}
