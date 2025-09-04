package main

import (
	"fmt"
	"os"
	
	cmd "github.com/liliang-cn/rago/v2/cmd/rago"
)

func main() {
	rootCmd := cmd.GetRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}