package main
import ("flag"
				"github.com/dubeyKartikay/lazyspotify/cli"
			ui "github.com/dubeyKartikay/lazyspotify/ui/v1"
)
func main() {
	flag.Parse()
	switch {
		case flag.NArg() > 0:
      cli.Run(flag.Args())
    default:
			ui.RunTui()
	}
}
