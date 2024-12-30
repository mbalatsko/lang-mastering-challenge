package services

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	CredDir      = ".tm-manager"
	CredFilename = "cred.json"
)

type Credentials struct {
	Token string `json:"token"`
}

func ParseCredentials(body []byte) (*Credentials, error) {
	credentials := &Credentials{}
	err := json.Unmarshal(body, &credentials)
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

func (c *Credentials) Save() error {
	homeDir := os.Getenv("HOME")
	err := os.MkdirAll(filepath.Join(homeDir, CredDir), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
