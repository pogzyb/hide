package main

import (
	"github.com/pogzyb/hide/cmd"
	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

func main() { cmd.Execute() }
