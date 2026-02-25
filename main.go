package main

import (
	"math/rand"

	sdk "github.com/ai-goblins/goblin-sdk"
)

func main() {
	input, err := sdk.ReadInput()
	if err != nil {
		sdk.WriteError(err)
		return
	}

	output, err := run(input, input.RunAt, rand.Intn)
	if err != nil {
		sdk.WriteError(err)
		return
	}

	sdk.WriteOutput(output)
}
