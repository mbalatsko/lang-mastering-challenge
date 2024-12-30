package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	AddrPing string

	PingCmd = &cobra.Command{
		Use:   "ping",
		Short: "Check if web application is online",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := http.Get(fmt.Sprintf("%s/ping", AddrPing))
			if err != nil {
				fmt.Printf("failed to reach out server: %s\n", err.Error())
				os.Exit(1)
			}

			body, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()

			if err != nil {
				fmt.Printf("failed to read response from server: %s\n", err.Error())
				os.Exit(1)
			}

			if string(body) == "pong" {
				fmt.Print("Success!")
			} else {
				fmt.Printf("error! Server responded with: %s\n", string(body))
				os.Exit(1)
			}
		},
	}
)

func init() {
	RootCmd.AddCommand(PingCmd)
}
