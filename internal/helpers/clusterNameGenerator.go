package helpers

import (
	"fmt"
	"math/rand"
	"time"
)

var adjectives = []string{
	"brave", "silent", "swift", "bright", "mighty", "gentle", "fierce",
	"bold", "fearless", "noble", "epic", "wild", "calm", "stormy",
	"radiant", "shadow", "cosmic", "ancient", "quick", "silent", "valiant",
	"clever", "fiery", "frozen", "golden", "iron", "lunar", "stellar",
	"mystic", "roaring", "shiny", "thunder", "wise", "zen",
}

var nouns = []string{
	"falcon", "tiger", "nebula", "phoenix", "dragon", "wolf", "eagle",
	"lion", "panther", "hawk", "orca", "viper", "bear", "rhino",
	"galaxy", "comet", "meteor", "star", "planet", "unicorn",
	"serpent", "griffin", "kraken", "leviathan", "pegasus",
	"titan", "giant", "wizard", "ninja", "samurai", "monk",
	"guardian", "phantom", "shadow", "knight", "sage",
}

func GenerateProjectName() string {
	rand.Seed(time.Now().UnixNano()) // ensure randomness each run

	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	num := rand.Intn(1000) // optional number for uniqueness

	return fmt.Sprintf("%s-%s-%d", adj, noun, num)
}
