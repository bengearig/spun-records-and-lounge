package main

import (
	"os"

	"github.com/bengearig/spun-records-and-lounge/internal/cli"
	_ "github.com/bengearig/spun-records-and-lounge/internal/features/costs"
	_ "github.com/bengearig/spun-records-and-lounge/internal/features/math"
)

func main() {
	os.Exit(cli.Execute())
}
