package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"tm-client/services"

	"github.com/spf13/cobra"
)

var (
	AddrLogin     string
	EmailLogin    string
	PasswordLogin string

	LoginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login to Task Manager application.",
		Long:  "Login to Task Manager application. On successful login token will be stored in $HOME/.tm-manager/cred.json",
		Run: func(cmd *cobra.Command, args []string) {
			userMap := map[string]string{
				"email":    EmailLogin,
				"password": PasswordLogin,
			}
			userJson, err := json.Marshal(userMap)
			if err != nil {
				fmt.Printf("failed to marshall user json: %s\n", err.Error())
				os.Exit(1)
			}

			resp, err := http.Post(fmt.Sprintf("%s/auth/login", AddrLogin), "application/json", bytes.NewBuffer(userJson))
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

			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Server rejected login with %d status and body: %s\n", resp.StatusCode, string(body))
				os.Exit(33)
			}

			cred, err := services.ParseCredentials(body)
			if err != nil {
				fmt.Printf("Failed to parse token from server response with %d status and body: %s, error: %s\n", resp.StatusCode, string(body), err.Error())
				os.Exit(1)
			}

			err = cred.Save()
			if err != nil {
				fmt.Printf("Failed to save credentials, error: %s\n", err.Error())
				os.Exit(1)
			}

			fmt.Println("Login was successful!")
		},
	}
)

func init() {
	LoginCmd.Flags().StringVarP(&AddrLogin, "address", "a", "", "Web application address (required)")
	LoginCmd.MarkFlagRequired("address")

	LoginCmd.Flags().StringVarP(&EmailLogin, "email", "e", "", "New user email (required)")
	LoginCmd.MarkFlagRequired("email")

	LoginCmd.Flags().StringVarP(
		&PasswordLogin,
		"password",
		"p",
		os.Getenv("TM_PASSWORD"),
		"New user password, could be passed. Could be passed as environment variable `TM_PASSWORD` (required)",
	)

	RootCmd.AddCommand(LoginCmd)
}
