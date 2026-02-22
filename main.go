package main

import (
	"math/rand"
	"time"

	sdk "github.com/ai-goblins/goblin-sdk"
)

func main() {
	input, err := sdk.ReadInput()
	if err != nil {
		sdk.WriteError(err)
		return
	}

	output, err := run(input, time.Now().UTC(), rand.Intn)
	if err != nil {
		sdk.WriteError(err)
		return
	}

	sdk.WriteOutput(output)
}
