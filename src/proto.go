package proxy

import (
	"encoding/json"
	"fmt"
)

type JSONRpcResp struct {
	Id     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
}

type StratumReq struct {
	JSONRpcResp
	Worker string `json:"worker"`
}

func (s *StratumReq) String() string {
	m, err := s.Id.MarshalJSON()
	if err != nil {
		return ""
	}
	id := string(m)
	p, err := s.Params.MarshalJSON()
	if err != nil {
		return ""
	}
	params := string(p)
	return fmt.Sprintf(`{"id": %s,"method": %s,"params": %s}`, id, s.Method, params)
}

type JSONResponse struct {
	Id      *json.RawMessage `json:"id"`
	Version string           `json:"version"`
	Result  interface{}      `json:"result"`
	Error   *ErrorReply      `json:"error"`
}

func (s *JSONResponse) String() string {
	m, err := s.Id.MarshalJSON()
	if err != nil {
		return ""
	}
	id := string(m)
	return fmt.Sprintf(`{"id": %s,"version": %s,"result": %s,"error": %s}`, id, s.Version, s.Result.([]string), s.Error.String())
}

type ErrorReply struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *ErrorReply) String() string {
	return fmt.Sprintf(`{"code": %d,"message": %s}`, s.Code, s.Message)
}
