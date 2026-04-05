package v1

type Cassette struct {
	spokeLeft Spoke
	spokeRight Spoke
}

func NewCassette() Cassette {
	return Cassette{
		spokeLeft: NewSpoke(),
		spokeRight: NewSpoke(),
	}
}

func (c *Cassette) View(){


}
