package pkg

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func _CreateFile(path string, mode int64) error {
	info, err := os.Stat(path)
	if nil == err {
		if info.IsDir() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		return nil
	}
	directory, filename := filepath.Split(path)
	err = os.MkdirAll(directory, os.ModePerm)
	if nil == err {
		//file, err := os.OpenFile(file, os.O_RDONLY|os.O_CREATE, 0644)
		//file, err := os.OpenFile(file, os.O_RDONLY|os.O_CREATE, os.FileMode(mode))
		//file, err := os.Create(path)
		file, err := os.OpenFile(filepath.Join(directory, filename), os.O_RDONLY|os.O_CREATE, os.FileMode(mode))
		if err != nil {
			return err
		}
		return file.Close()
	}
	return nil
}

func _SaveFile(mode int64, name string, content []byte) error {
	return ioutil.WriteFile(name, content, os.FileMode(mode))
}

func Download(source, destination string, host, user, password string) error {
	if err := _CreateFile(destination, int64(os.ModePerm)); nil != err {
		return err
	}
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
	reader, err := session.StdoutPipe()
	if nil != err {
		return err
	}
	writer, err := session.StdinPipe()
	if nil != err {
		return err
	}
	err = session.Start(fmt.Sprintf("scp -fv %s", source))
	if nil != err {
		return err
	}
	errors := make(chan error)
	go func() {
		errors <- session.Wait()
	}()
	_, err = writer.Write([]byte("\x00"))
	if nil != err {
		return err
	}
	{
		// Read Incoming bytes
		reader := bufio.NewReader(reader)
		str, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read intitial file header: %w", err)
		} else if len(str) == 0 {
			return fmt.Errorf("empty request")
		}

		str = str[1:]
		fields := strings.Fields(str)
		if len(fields) != 3 {
			return fmt.Errorf("protocol demands 3 fields, got %d", len(fields))
		}

		mode, err := strconv.ParseInt(fields[0], 8, 32)
		if err != nil {
			return fmt.Errorf("failed to parse the mode: %q (%w)", fields[0], err)
		}

		length, err := strconv.ParseInt(fields[1], 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse the length: %q (%w)", fields[1], err)
		}
		// filename := fields[2]
		_, err = writer.Write([]byte("\x00"))
		if nil != err {
			return err
		}
		contents := make([]byte, length+1)
		read, err := io.ReadFull(reader, contents)
		if nil != err {
			return err
		}
		if int64(read) != length+1 {
			return fmt.Errorf("short read, want %d bytes but got %d", length+1, read)
		}
		contents = contents[:len(contents)-1]
		// _SaveFile(mode, filename, contents)
		err = _SaveFile(mode, destination, contents)
		if nil != err {
			return err
		}
	}
	_, err = writer.Write([]byte("\x00"))
	if nil != err {
		return err
	}
	_ = writer.Close()
	err = <-errors
	return err
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
	_ = writer.Close()
	err = <-errors
	return err
}
