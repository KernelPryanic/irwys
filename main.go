package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/FromZeus/irwys/irwys"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	minLength = kingpin.Flag(
		"minLength",
		"Minimal operational message lenght (in words). Min: 0",
	).Default("4").Uint16()
	maxLength = kingpin.Flag(
		"maxLength",
		fmt.Sprintf("Maximal operational message lenght (in words). Max: %d", math.MaxUint16),
	).Default(strconv.FormatUint(math.MaxUint16, 10)).Uint16()
	timeout = kingpin.Flag(
		"timeout",
		"How long to wait after last message was posted (in minutes).",
	).Default("10").Short('t').Int16()
	capacity = kingpin.Flag(
		"capacity",
		"Capacity of message storage per chat (in messages).",
	).Default("2048").Short('c').Uint16()
	dbPath = kingpin.Flag(
		"dbPath",
		"Path to level db.",
	).Default("./db").Short('d').String()
	replyPath = kingpin.Flag(
		"replyPath",
		"Path to reply dictionaries.",
	).Default("./replies").Short('r').String()
	verbose = kingpin.Flag(
		"verbose",
		"Verbose logging mode.",
	).Short('v').Bool()
	token = kingpin.Arg(
		"token",
		"Bot's token.",
	).Required().String()
)

func main() {
	kingpin.Parse()
	opts := irwys.NewOptions(
		*minLength,
		*maxLength,
		*timeout,
		*capacity,
		*dbPath,
		*replyPath,
		*verbose,
	)

	bot := irwys.New(*token, &opts)
	bot.Start()
}
