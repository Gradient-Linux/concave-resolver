package resolver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ServeSocket starts the Unix socket server and handles JSON requests.
func ServeSocket(ctx context.Context, socketPath string, service *Service) error {
	if service == nil {
		return fmt.Errorf("service is nil")
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale socket: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	defer listener.Close()
	defer os.Remove(socketPath)

	if err := os.Chmod(socketPath, 0o660); err != nil {
		return fmt.Errorf("set socket mode: %w", err)
	}

	acceptErr := make(chan error, 1)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				acceptErr <- err
				return
			}
			go handleConnection(conn, service)
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-acceptErr:
		if err == nil || errors.Is(err, net.ErrClosed) {
			return nil
		}
		return err
	}
}

// QueryStatus sends a status request to the resolver socket.
func QueryStatus(socketPath string) (ResolverStatus, error) {
	var response struct {
		Type    string         `json:"type"`
		Payload ResolverStatus `json:"payload"`
		Error   string         `json:"error"`
	}
	if err := querySocket(socketPath, StatusRequest{Type: "status"}, &response); err != nil {
		return ResolverStatus{}, err
	}
	if response.Error != "" {
		return ResolverStatus{}, errors.New(response.Error)
	}
	return response.Payload, nil
}

// QueryDrift sends a drift request to the resolver socket.
func QueryDrift(socketPath, group string) ([]DriftReport, error) {
	var response struct {
		Type    string        `json:"type"`
		Payload []DriftReport `json:"payload"`
		Error   string        `json:"error"`
	}
	if err := querySocket(socketPath, DriftRequest{Type: "drift", Group: group}, &response); err != nil {
		return nil, err
	}
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	if response.Payload == nil {
		return []DriftReport{}, nil
	}
	return response.Payload, nil
}

func handleConnection(conn net.Conn, service *Service) {
	defer conn.Close()

	request, err := readRequest(conn)
	if err != nil {
		_ = writeResponse(conn, Response{Type: "error", Error: err.Error()})
		return
	}

	switch request.Type {
	case "status":
		_ = writeResponse(conn, Response{Type: "status", Payload: service.Status()})
	case "drift":
		_ = writeResponse(conn, Response{Type: "drift", Payload: service.DriftReports(request.Group)})
	case "baseline_set":
		_ = writeResponse(conn, Response{Type: "error", Error: "baseline promotion is not implemented in the scaffold"})
	default:
		_ = writeResponse(conn, Response{Type: "error", Error: "unknown request type"})
	}
}

type socketRequest struct {
	Type      string `json:"type"`
	Group     string `json:"group,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

func readRequest(conn net.Conn) (socketRequest, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return socketRequest{}, fmt.Errorf("read request: %w", err)
	}

	var envelope socketRequest
	if err := json.Unmarshal(line, &envelope); err != nil {
		return socketRequest{}, fmt.Errorf("decode request: %w", err)
	}

	switch envelope.Type {
	case "status":
		var request StatusRequest
		if err := json.Unmarshal(line, &request); err != nil {
			return socketRequest{}, fmt.Errorf("decode status request: %w", err)
		}
		return socketRequest{Type: request.Type}, nil
	case "drift":
		var request DriftRequest
		if err := json.Unmarshal(line, &request); err != nil {
			return socketRequest{}, fmt.Errorf("decode drift request: %w", err)
		}
		return socketRequest{Type: request.Type, Group: request.Group}, nil
	case "baseline_set":
		var request ApplyBaselineRequest
		if err := json.Unmarshal(line, &request); err != nil {
			return socketRequest{}, fmt.Errorf("decode baseline request: %w", err)
		}
		return socketRequest{Type: request.Type, Group: request.Group, Timestamp: request.Timestamp}, nil
	default:
		return socketRequest{}, fmt.Errorf("unknown request type %q", envelope.Type)
	}
}

func writeResponse(conn net.Conn, response Response) error {
	encoder := json.NewEncoder(conn)
	return encoder.Encode(response)
}

func querySocket(socketPath string, request any, response any) error {
	conn, err := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
	if err != nil {
		return fmt.Errorf("dial socket: %w", err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	if err := json.NewDecoder(conn).Decode(response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
