package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/csnewman/dyndirect/cli/proxy"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dyndirect",
		Short: "dyn.direct tool",
		Long:  "dyndirect: CLI client for interacting with https://dyn.direct.",
	}

	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "Start a HTTPS proxy",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			src, err := strconv.ParseInt(args[0], 10, 16)
			if err != nil {
				fmt.Println("Invalid source port") //nolint:forbidigo
				os.Exit(1)
			}

			host, _ := cmd.Flags().GetBool("override-host")
			replace, _ := cmd.Flags().GetBool("replace")
			proxy.RunProxy(int(src), args[1], host, replace)
		},
	}
	proxyCmd.Flags().Bool("override-host", false, "Overwrite Host header")
	proxyCmd.Flags().Bool("replace", false, "Replace subdomain and certificate")

	rootCmd.AddCommand(proxyCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err) //nolint:forbidigo
		os.Exit(1)
	}
}
