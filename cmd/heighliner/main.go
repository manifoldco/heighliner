package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:    "heighliner",
	Hidden: true,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("heighliner error: %s\n", err)
	}
}
