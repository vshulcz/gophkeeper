// Command gophkeeper-client provides the CLI client.
package main

import (
	"gophkeeper/internal/app/cli"
	"log"
	"os"
)

func main() {
	if err := cli.Run(os.Args[1:], cli.DefaultDeps(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}
