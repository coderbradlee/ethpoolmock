package proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	MaxReqSize = 1024
	headerHash = "645cf20198c2f3861e947d4f67e3ab63b7b2e24dcc9095bd9123e7b33371f6cc"
	seedHash   = "abad8f99f3918bf903c6a909d9bbc0fdfa5a2f4b9cb1196175ec825c6610126c"
	diff       = "2181977828349372"
)

type ProxyServer struct {
	sessionsMu sync.RWMutex
	sessions   map[*Session]struct{}
}

type Session struct {
	ip  string
	enc *json.Encoder
	sync.Mutex
	conn  *net.TCPConn
	login string
}

func NewProxy() {
	proxy := &ProxyServer{}
	proxy.sessions = make(map[*Session]struct{})
	proxy.ListenTCP()
}

func (s *ProxyServer) ListenTCP() {
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:8080")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	server, err := net.ListenTCP("tcp", addr)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	defer server.Close()

	fmt.Println("Stratum listening on 0.0.0.0:8080")
	var accept = make(chan int, 100)
	n := 0
	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			continue
		}
		fmt.Println("have conn", conn)
		conn.SetKeepAlive(true)

		ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		n += 1
		cs := &Session{conn: conn, ip: ip}

		accept <- n
		go func(cs *Session) {
			err = s.handleTCPClient(cs)
			if err != nil {
				s.removeSession(cs)
				conn.Close()
			}
			<-accept
		}(cs)
	}
}

func (s *ProxyServer) handleTCPClient(cs *Session) error {
	cs.enc = json.NewEncoder(cs.conn)
	connbuff := bufio.NewReaderSize(cs.conn, MaxReqSize)
	s.setDeadline(cs.conn)

	for {
		data, isPrefix, err := connbuff.ReadLine()
		if isPrefix {
			fmt.Printf("Socket flood detected from %s\n", cs.ip)
			return err
		} else if err == io.EOF {
			fmt.Printf("Client %s disconnected\n", cs.ip)
			s.removeSession(cs)
			break
		} else if err != nil {
			fmt.Printf("Error reading from socket: %v\n", err)
			return err
		}

		if len(data) > 1 {
			var req StratumReq
			err = json.Unmarshal(data, &req)
			if err != nil {
				fmt.Printf("Malformed stratum request from %s: %v\n", cs.ip, err)
				return err
			}
			s.setDeadline(cs.conn)
			err = cs.handleTCPMessage(s, &req)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cs *Session) handleTCPMessage(s *ProxyServer, req *StratumReq) error {
	fmt.Println("handleTCPMessage request:", req)
	// Handle RPC methods
	switch req.Method {
	case "eth_submitLogin":
		var params []string
		mar, err := req.Params.MarshalJSON()
		if err != nil {
			fmt.Println("handleTCPMessage MarshalJSON", err)
		}
		err = json.Unmarshal(mar, &params)
		if err != nil {
			fmt.Println("Malformed stratum request params from", cs.ip)
			return err
		}
		reply, errReply := s.handleLoginRPC(cs, params, req.Worker)
		if errReply != nil {
			return cs.sendTCPError(req.Id, errReply)
		}

		return cs.sendTCPResult(req.Id, reply)
	case "eth_getWork":
		reply, errReply := s.handleGetWorkRPC(cs)
		if errReply != nil {
			return cs.sendTCPError(req.Id, errReply)
		}
		return cs.sendTCPResult(req.Id, &reply)
	case "eth_submitWork":
		var params []string
		mar, err := req.Params.MarshalJSON()
		if err != nil {
			fmt.Println("handleTCPMessage MarshalJSON", err)
		}
		err = json.Unmarshal(mar, &params)
		if err != nil {
			fmt.Println("Malformed stratum request params from", cs.ip)
			return err
		}
		reply, errReply := s.handleTCPSubmitRPC(cs, req.Worker, params)
		if errReply != nil {
			return cs.sendTCPError(req.Id, errReply)
		}
		return cs.sendTCPResult(req.Id, &reply)
	case "eth_submitHashrate":
		return cs.sendTCPResult(req.Id, true)
	default:
		errReply := s.handleUnknownRPC(cs, req.Method)
		return cs.sendTCPError(req.Id, errReply)
	}
}

func (cs *Session) sendTCPResult(id *json.RawMessage, result interface{}) error {
	cs.Lock()
	defer cs.Unlock()

	message := JSONResponse{Id: id, Version: "2.0", Error: nil, Result: result}
	fmt.Println("response:", message)
	return cs.enc.Encode(&message)
}

func (cs *Session) pushNewJob(result interface{}) error {
	cs.Lock()
	defer cs.Unlock()
	// FIXME: Temporarily add ID for Claymore compliance
	message := JSONResponse{Id: &json.RawMessage{1}, Version: "2.0", Result: result}
	fmt.Println("pushNewJob:", message)
	return cs.enc.Encode(&message)
}

func (cs *Session) sendTCPError(id *json.RawMessage, reply *ErrorReply) error {
	cs.Lock()
	defer cs.Unlock()

	message := JSONResponse{Id: id, Version: "2.0", Error: reply}
	err := cs.enc.Encode(&message)
	if err != nil {
		return err
	}
	return errors.New(reply.Message)
}

func (self *ProxyServer) setDeadline(conn *net.TCPConn) {
	conn.SetDeadline(time.Now().Add(time.Second * 10))
}

func (s *ProxyServer) registerSession(cs *Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[cs] = struct{}{}
}

func (s *ProxyServer) removeSession(cs *Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	delete(s.sessions, cs)
}

func (s *ProxyServer) broadcastNewJobs() {
	reply := []string{seedHash, headerHash, diff}

	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	count := len(s.sessions)
	fmt.Printf("Broadcasting new job to %v stratum miners\n", count)

	start := time.Now()
	bcast := make(chan int, 1024)
	n := 0

	for m, _ := range s.sessions {
		n++
		bcast <- n

		go func(cs *Session) {
			err := cs.pushNewJob(&reply)
			<-bcast
			if err != nil {
				fmt.Printf("Job transmit error to %v@%v: %v\n", cs.login, cs.ip, err)
				s.removeSession(cs)
			} else {
				s.setDeadline(cs.conn)
			}
		}(m)
	}
	fmt.Printf("Jobs broadcast finished %s\n", time.Since(start))
}
