package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"path/filepath"
)

// config is set in config.json
type config struct {
	Server     string `json:"Server"`
	User       string `json:"User"`
	Password   string `json:"Password"`
	SourcePath string `json:"SourcePath"`
	DstPath    string `json:"DstPath"`
}

func main() {
	// read configs
	var err error
	content, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(err.Error())
	}
	var cfg config
	err = json.Unmarshal(content, &cfg)
	if err != nil {
		log.Fatal("Parse config failed. Ensure you have the config.json file correctly set up. ")
	}

	client, err := ssh.Dial("tcp", cfg.Server, &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password(cfg.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		log.Fatalf("SSH dial error: %s", err.Error())
	}

	// open an SFTP session over an existing ssh connection.
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		panic(err)
	}
	defer sftpClient.Close()

	srcPath := cfg.SourcePath
	dstPath := cfg.DstPath
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		panic(err)
	}
	// check is file or directory
	fileInfo, err := srcFile.Stat()
	if err != nil {
		panic(err)
	}
	// zip the directory if selected directory
	if fileInfo.IsDir() {
		zipFileName := fileInfo.Name() + ".zip"
		err = zipSource(cfg.SourcePath, zipFileName)
		if err != nil {
			log.Fatal("zipping directory failed. ")
		}
		// clean up temporary local zip file
		defer os.Remove(zipFileName)
		// change source file to the zip file
		srcFile, err = os.Open(zipFileName)
		if err != nil {
			log.Fatal(err.Error())
		}
		// change target file from directory tp zip file
		dstPath += zipFileName
	} else {
		dstPath += fileInfo.Name()
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := sftpClient.Create(dstPath)
	if err != nil {
		panic(err)
	}
	defer dstFile.Close()

	// write to file
	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		log.Fatal(err)
	}

	// if created zip file. unzip it and clean up.
	if fileInfo.IsDir() {
		// unzip to a directory
		err = Exec(client, fmt.Sprintf(" unzip %s", dstPath))
		if err != nil {
			log.Fatal(err)
		}
		// delete zip file
		err = Exec(client, fmt.Sprintf("rm %s", fileInfo.Name()+".zip"))
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

func zipSource(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}

// Exec execute a command on ssh client and report error if any.
func Exec(client *ssh.Client, cmd string) error {
	// 建立新会话
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	// delete zip file
	err = session.Run(cmd)
	if err != nil {
		return err
	}
	return nil
}
