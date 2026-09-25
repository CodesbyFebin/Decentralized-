// dh is the Decentralized.Host operator CLI.
package main

import (
	"os"

	_ "decentralized.host/pkg/chaos" // chaos list / run / soak
	"decentralized.host/pkg/cli"
	_ "decentralized.host/pkg/devcluster" // dev up / down / status
	_ "decentralized.host/pkg/evidence"   // evidence env / run / seal / verify
)

func main() { os.Exit(cli.Main(os.Args[1:])) }
