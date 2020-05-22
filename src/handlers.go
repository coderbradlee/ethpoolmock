package proxy

import (
	"fmt"
	"strings"
)

// Stratum
func (s *ProxyServer) handleLoginRPC(cs *Session, params []string, id string) (bool, *ErrorReply) {
	if len(params) == 0 {
		return false, &ErrorReply{Code: -1, Message: "Invalid params"}
	}

	login := strings.ToLower(params[0])
	cs.login = login
	s.registerSession(cs)
	fmt.Printf("Stratum miner connected %s@%s\n", login, cs.ip)
	return true, nil
}

func (s *ProxyServer) handleGetWorkRPC(cs *Session) ([]string, *ErrorReply) {
	return []string{seedHash, headerHash, diff}, nil
}

// Stratum
func (s *ProxyServer) handleTCPSubmitRPC(cs *Session, id string, params []string) (bool, *ErrorReply) {
	s.sessionsMu.RLock()
	_, ok := s.sessions[cs]
	s.sessionsMu.RUnlock()

	if !ok {
		return false, &ErrorReply{Code: 25, Message: "Not subscribed"}
	}
	return s.handleSubmitRPC(cs, cs.login, id, params)
}

func (s *ProxyServer) handleSubmitRPC(cs *Session, login, id string, params []string) (bool, *ErrorReply) {
	return true, nil
}

func (s *ProxyServer) handleUnknownRPC(cs *Session, m string) *ErrorReply {
	fmt.Printf("Unknown request method %s from %s\n", m, cs.ip)
	return &ErrorReply{Code: -3, Message: "Method not found"}
}
