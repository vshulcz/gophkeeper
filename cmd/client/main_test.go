package main

import (
	"os"
	"testing"
)

func TestMain_NoArgs(t *testing.T) {
	t.Parallel()
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"gophkeeper"}
	main()
}
