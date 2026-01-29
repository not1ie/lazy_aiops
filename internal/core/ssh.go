package core

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient SSH 客户端配置
type SSHClient struct {
	Host     string
	Port     int
	Username string
	Password string
	Key      string
	Timeout  time.Duration
}

// Execute 执行远程命令
func (s *SSHClient) Execute(command string) (string, string, error) {
	config := &ssh.ClientConfig{
		User: s.Username,
		Auth: []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         s.Timeout,
	}

	if s.Password != "" {
		config.Auth = append(config.Auth, ssh.Password(s.Password))
	}
	if s.Key != "" {
		signer, err := ssh.ParsePrivateKey([]byte(s.Key))
		if err == nil {
			config.Auth = append(config.Auth, ssh.PublicKeys(signer))
		}
	}

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	conn, err := net.DialTimeout("tcp", addr, config.Timeout)
	if err != nil {
		return "", "", err
	}
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return "", "", err
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	return stdout.String(), stderr.String(), err
}
