package publishing

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const defaultSSHPort = "22"

type SSHDialer struct {
	Timeout time.Duration
}

func (d SSHDialer) Dial(ctx context.Context, cfg SSHConfig) (Client, error) {
	key, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read SSH key: %w", err)
	}

	var signer ssh.Signer
	if cfg.Passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(cfg.Passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		return nil, fmt.Errorf("parse SSH key: %w", err)
	}

	timeout := d.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	address := withDefaultPort(cfg.Host)
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connect SSH: %w", err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("handshake SSH: %w", err)
	}
	sshClient := ssh.NewClient(sshConn, chans, reqs)

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, fmt.Errorf("start SFTP: %w", err)
	}
	return &client{
		Client:    sftpClient,
		sshClient: sshClient,
	}, nil
}

type client struct {
	*sftp.Client
	sshClient *ssh.Client
}

func (c *client) Create(path string) (RemoteFile, error) {
	return c.Client.Create(path)
}

func (c *client) Close() error {
	sftpErr := c.Client.Close()
	sshErr := c.sshClient.Close()
	if sftpErr != nil {
		return sftpErr
	}
	return sshErr
}

func withDefaultPort(host string) string {
	host = strings.TrimSpace(host)
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, defaultSSHPort)
}
