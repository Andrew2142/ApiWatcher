package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
)

// Server handles incoming control connections
type Server struct {
	daemon   *Daemon
	listener net.Listener
	address  string
	stopChan chan bool
	wg       sync.WaitGroup
}

// NewServer creates a new control server
func NewServer(daemon *Daemon, address string) *Server {
	return &Server{
		daemon:   daemon,
		address:  address,
		stopChan: make(chan bool),
	}
}

// Start starts the control server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.listener = listener
	log.Printf("Control server listening on %s", s.address)

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop stops the control server
func (s *Server) Stop() {
	close(s.stopChan)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	log.Println("Control server stopped")
}

// acceptConnections accepts incoming connections
func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	log.Printf("Client connected: %s", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	encoder := json.NewEncoder(conn)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse command
		var cmd Command
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			response := Response{
				Success: false,
				Message: fmt.Sprintf("invalid command format: %v", err),
			}
			encoder.Encode(response)
			continue
		}

		// Handle command
		response := s.daemon.HandleCommand(cmd)

		// Send response
		if err := encoder.Encode(response); err != nil {
			log.Printf("Failed to send response: %v", err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Connection error: %v", err)
	}

	log.Printf("Client disconnected: %s", conn.RemoteAddr())
}

