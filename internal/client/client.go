package client

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func New(address string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Set(key, value string, ttl time.Duration) error {
	cmd := fmt.Sprintf("SET %s %s", key, value)
	if ttl > 0 {
		cmd += fmt.Sprintf(" %s", ttl)
	}
	response, err := c.executeCommand(cmd)
	if err != nil {
		return err
	}

	if response != "OK" {
		return fmt.Errorf("unexpected response: %s", response)
	}
	return nil
}

func (c *Client) Get(key string) (string, error) {
    response, err := c.executeCommand(fmt.Sprintf("GET %s", key))
    if err != nil {
        return "", err
    }
    
    if response == "NIL" {
        return "", nil
    }
    return response, nil
}

func (c *Client) Delete(key string) error {
	response, err := c.executeCommand(fmt.Sprintf("DELETE %s", key))
	if err != nil {
		return err
	}

	if response != "OK" {
		return fmt.Errorf("unexptected response %s", response)
	}
	return nil
}

func (c *Client) Exists(key string) (bool, error) {
    response, err := c.executeCommand(fmt.Sprintf("EXISTS %s", key))
    if err != nil {
        return false, err
    }
    
    return response == "1", nil
}

func (c *Client) executeCommand(cmd string) (string, error) {
	_, err := c.writer.WriteString(cmd + "\n")
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	if err := c.writer.Flush(); err != nil {
		return "", fmt.Errorf("failed to flush command: %w", err)
	}

	response, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return strings.TrimSpace(response), nil
}
