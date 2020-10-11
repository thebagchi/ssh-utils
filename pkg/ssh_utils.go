package pkg

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path/filepath"
)

func _MakeConfig(user, password string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func _Connect(host string, config *ssh.ClientConfig) (*ssh.Client, error) {
	return ssh.Dial("tcp", host, config)
}

func RunCommand(host, user, password string, command string) ([]byte, error) {
	config := _MakeConfig(user, password)
	conn, err := _Connect(host, config)
	if nil != err {
		return nil, err
	}
	defer conn.Close()
	_, _, err = conn.SendRequest("keepalive@openssh.com", true, nil)
	session, err := conn.NewSession()
	if nil != err {
		return nil, err
	}
	defer session.Close()
	var buffer bytes.Buffer
	session.Stdout = &buffer
	err = session.Start(command)
	if nil != err {
		return nil, err
	}
	errors := make(chan error)
	go func() {
		errors <- session.Wait()
	}()
	err = <-errors
	return buffer.Bytes(), err
}

func Download(source, destination string, host, user, password string) error {
	config := _MakeConfig(user, password)
	conn, err := _Connect(host, config)
	if nil != err {
		return err
	}
	defer conn.Close()
	_, _, err = conn.SendRequest("keepalive@openssh.com", true, nil)
	session, err := conn.NewSession()
	if nil != err {
		return err
	}
	defer session.Close()
	return nil
}

func Upload(source, destination string, host, user, password string) error {
	info, err := os.Stat(source)
	if nil != err {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is not a regular file", source)
	}
	file, err := os.Open(source)
	if nil != err {
		return err
	}
	defer file.Close()
	var (
		mode = info.Mode()
		size = info.Size()
	)
	config := _MakeConfig(user, password)
	conn, err := _Connect(host, config)
	if nil != err {
		return err
	}
	defer conn.Close()
	_, _, err = conn.SendRequest("keepalive@openssh.com", true, nil)
	session, err := conn.NewSession()
	if nil != err {
		return err
	}
	defer session.Close()
	writer, err := session.StdinPipe()
	if nil != err {
		return err
	}
	err = session.Start(fmt.Sprintf("scp -tv %s", filepath.Dir(destination)))
	if nil != err {
		return err
	}
	errors := make(chan error)
	go func() {
		errors <- session.Wait()
	}()
	_, err = writer.Write([]byte(fmt.Sprintf("C0%o %d %s\n", mode, size, filepath.Base(destination))))
	if nil != err {
		return err
	}
	_, err = io.Copy(writer, file)
	if nil != err {
		return err
	}
	_, err = writer.Write([]byte("\x00"))
	if nil != err {
		return err
	}
	writer.Close()
	err = <-errors
	return err
}
