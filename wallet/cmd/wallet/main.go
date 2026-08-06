package main

import (
	"fmt"
	"os"

	"github.com/escalopa/minichain/wallet/internal/cli"
)

func main() {
	if err := cli.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
