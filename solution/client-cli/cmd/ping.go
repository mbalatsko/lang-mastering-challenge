package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	Addr string

	PingCmd = &cobra.Command{
		Use:   "ping",
		Short: "Check if web application is online",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := http.Get(fmt.Sprintf("%s/ping", Addr))
			if err != nil {
				fmt.Printf("failed to reach out server: %s", err.Error())
				os.Exit(1)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("failed to read response from server: %s", err.Error())
				os.Exit(1)
			}

			if string(body) == "pong" {
				fmt.Print("Success!")
			} else {
				fmt.Printf("error! Server responded with: %s", string(body))
				os.Exit(1)
			}
		},
	}
)

func init() {
	PingCmd.Flags().StringVarP(&Addr, "address", "a", "", "Web application address (required)")
	PingCmd.MarkFlagRequired("address")

	rootCmd.AddCommand(PingCmd)
}
