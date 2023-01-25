package main

import (
	"encoding/json"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
)

type config struct {
	Server     string `json:"Server"`
	User       string `json:"User"`
	Password   string `json:"Password"`
	SourcePath string `json:"SourcePath"`
	DstPath    string `json:"DstPath"`
}

func main() {
	// read configs
	content, err := os.ReadFile("config.json")
	var cfg config
	json.Unmarshal(content, &cfg)

	client, err := ssh.Dial("tcp", cfg.Server, &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password(cfg.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		log.Fatalf("SSH dial error: %s", err.Error())
	}

	// 建立新会话
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("new session error: %s", err.Error())
	}

	defer session.Close()

	// open an SFTP session over an existing ssh connection.
	sftp, err := sftp.NewClient(client)
	if err != nil {
		panic(err)
	}
	defer sftp.Close()

	srcPath := cfg.SourcePath
	dstPath := cfg.DstPath
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		panic(err)
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := sftp.Create(dstPath)
	if err != nil {
		panic(err)
	}
	defer dstFile.Close()

	// write to file
	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		panic(err)
	}
}
