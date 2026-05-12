package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient wraps a single SSH connection.
type SSHClient struct {
	client *ssh.Client
}

// Connect establishes an SSH connection.
func (s *SSHClient) Connect(host string, port int, username, authType, password, keyPath string,
	timeout time.Duration) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	config := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	switch authType {
	case "password":
		config.Auth = []ssh.AuthMethod{ssh.Password(password)}
	case "key":
		if keyPath == "" {
			return errors.New("密钥文件路径不能为空")
		}
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("读取密钥文件失败: %w", err)
		}
		var signer ssh.Signer
		signer, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			// Try with passphrase
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(password))
			if err != nil {
				return fmt.Errorf("解析密钥失败: %w", err)
			}
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	default:
		return fmt.Errorf("不支持的认证类型: %s", authType)
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	s.client = client
	return nil
}

// IsConnected returns whether the client is connected.
func (s *SSHClient) IsConnected() bool {
	return s.client != nil
}

// TestConnection quickly tests if a connection is valid.
func (s *SSHClient) TestConnection(conn *Connection, timeoutSec int) (bool, string) {
	c := &SSHClient{}
	err := c.Connect(conn.Host, conn.Port, conn.Username, conn.AuthType, conn.Password, conn.KeyPath, time.Duration(timeoutSec)*time.Second)
	if err == nil {
		c.Close()
		return true, "连接成功"
	}
	c.Close()
	return false, err.Error()
}

// Execute runs a command via SSH with line-by-line callback.
func (s *SSHClient) Execute(command string, lineCb func(line, stream string)) (string, string, int, error) {
	if s.client == nil {
		return "", "", -1, errors.New("SSH未连接")
	}

	session, err := s.client.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("创建会话失败: %w", err)
	}
	defer session.Close()

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return "", "", -1, err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return "", "", -1, err
	}

	if err := session.Start(command); err != nil {
		return "", "", -1, fmt.Errorf("执行命令失败: %w", err)
	}

	// Read stdout/stderr concurrently
	type readResult struct {
		data string
		err  error
	}
	stdoutCh := make(chan readResult, 1)
	stderrCh := make(chan readResult, 1)

	go func() {
		b, err := io.ReadAll(stdoutPipe)
		stdoutCh <- readResult{string(b), err}
	}()
	go func() {
		b, err := io.ReadAll(stderrPipe)
		stderrCh <- readResult{string(b), err}
	}()

	stdoutRes := <-stdoutCh
	stderrRes := <-stderrCh

	stdoutStr := stdoutRes.data
	stderrStr := stderrRes.data

	// Report to callback
	if lineCb != nil {
		if stdoutStr != "" {
			lineCb(stdoutStr, "stdout")
		}
		if stderrStr != "" {
			lineCb(stderrStr, "stderr")
		}
	}

	// Wait for completion
	err = session.Wait()
	exitCode := 0
	if exitErr, ok := err.(*ssh.ExitError); ok {
		exitCode = exitErr.ExitStatus()
	} else if err != nil {
		return stdoutStr, stderrStr, -1, err
	}

	return stdoutStr, stderrStr, exitCode, nil
}

// Close closes the SSH connection.
func (s *SSHClient) Close() {
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
}
