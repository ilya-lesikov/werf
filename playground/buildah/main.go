package main

import (
	"context"
	"fmt"
	"os"

	"github.com/werf/werf/pkg/buildah"
)

func main() {
	if err := do(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func do() error {
	return buildah.Run(context.Background(), "build-container", []string{"ls"}, buildah.NewRunInputOptions())
}
