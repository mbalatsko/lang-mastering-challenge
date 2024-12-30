package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	AddrReg     string
	EmailReg    string
	PasswordReg string

	RegisterCmd = &cobra.Command{
		Use:   "register",
		Short: "Register new user in Task Manager application.",
		Run: func(cmd *cobra.Command, args []string) {
			userMap := map[string]string{
				"email":    EmailReg,
				"password": PasswordReg,
			}
			userJson, err := json.Marshal(userMap)
			if err != nil {
				fmt.Printf("failed to marshall user json: %s\n", err.Error())
				os.Exit(1)
			}

			resp, err := http.Post(fmt.Sprintf("%s/auth/register", AddrLogin), "application/json", bytes.NewBuffer(userJson))
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

			if resp.StatusCode != http.StatusCreated {
				fmt.Printf("Server rejected registration with %d status and body: %s\n", resp.StatusCode, string(body))
				os.Exit(22)
			}

			fmt.Println("Registration successful!")
		},
	}
)

func init() {
	RegisterCmd.Flags().StringVarP(&AddrLogin, "address", "a", "", "Web application address (required)")
	RegisterCmd.MarkFlagRequired("address")

	RegisterCmd.Flags().StringVarP(&EmailReg, "email", "e", "", "New user email (required)")
	RegisterCmd.MarkFlagRequired("email")

	RegisterCmd.Flags().StringVarP(
		&PasswordReg,
		"password",
		"p",
		os.Getenv("TM_PASSWORD"),
		"New user password, could be passed. Could be passed as environment variable `TM_PASSWORD` (required)",
	)

	RootCmd.AddCommand(RegisterCmd)
}
