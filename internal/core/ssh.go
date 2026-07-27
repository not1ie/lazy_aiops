package core

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"log"
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

// pooledConn wraps an ssh.Client with a last-used timestamp
type pooledConn struct {
	client   *ssh.Client
	lastUsed time.Time
}

var (
	poolMu    sync.RWMutex
	clientMap = make(map[string]*pooledConn)
)

func init() {
	go func() {
		// Clean up idle connections every minute
		for {
			time.Sleep(1 * time.Minute)
			now := time.Now()
			poolMu.Lock()
			for addr, pConn := range clientMap {
				if now.Sub(pConn.lastUsed) > 5*time.Minute {
					pConn.client.Close()
					delete(clientMap, addr)
				}
			}
			poolMu.Unlock()
		}
	}()
}

func (s *SSHClient) getPooledClient() (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

	poolMu.RLock()
	if pConn, ok := clientMap[addr]; ok {
		// Quick check if connection is still alive
		_, _, err := pConn.client.SendRequest("keepalive@openssh.com", true, nil)
		if err == nil {
			log.Printf("[SSH Pool] Reusing connection to %s", addr)
			pConn.lastUsed = time.Now()
			poolMu.RUnlock()
			return pConn.client, nil
		}
		// Connection dead, close and delete
		pConn.client.Close()
		poolMu.RUnlock()

		poolMu.Lock()
		delete(clientMap, addr)
		poolMu.Unlock()
	} else {
		poolMu.RUnlock()
	}

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

	conn, err := net.DialTimeout("tcp", addr, config.Timeout)
	if err != nil {
		return nil, err
	}
	if config.Timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(config.Timeout))
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	client := ssh.NewClient(sshConn, chans, reqs)

	poolMu.Lock()
	clientMap[addr] = &pooledConn{
		client:   client,
		lastUsed: time.Now(),
	}
	poolMu.Unlock()

	return client, nil
}

// ExecuteWithPool 使用连接池执行远程命令
func (s *SSHClient) ExecuteWithPool(command string) (string, string, error) {
	client, err := s.getPooledClient()
	if err != nil {
		return "", "", err
	}

	session, err := client.NewSession()
	if err != nil {
		// If session creation fails, maybe the connection dropped. Force reconnect next time.
		addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
		poolMu.Lock()
		delete(clientMap, addr)
		poolMu.Unlock()
		client.Close()
		return "", "", err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	return stdout.String(), stderr.String(), err
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

	if config.Timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(config.Timeout))
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return "", "", err
	}
	_ = conn.SetDeadline(time.Time{})
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

// ExecuteStream 执行远程命令并将输出实时写入传入的 Writer
func (s *SSHClient) ExecuteStream(command string, stdoutWriter, stderrWriter io.Writer) error {
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
		return err
	}
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return err
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = stdoutWriter
	session.Stderr = stderrWriter

	return session.Run(command)
}
