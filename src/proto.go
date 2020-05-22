package proxy

import "encoding/json"

type JSONRpcResp struct {
	Id     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
}

type StratumReq struct {
	JSONRpcResp
	Worker string `json:"worker"`
}

type JSONResponse struct {
	Id      *json.RawMessage `json:"id"`
	Version string           `json:"jsonrpc"`
	Result  interface{}      `json:"result"`
	Error   *ErrorReply      `json:"error"`
}

type ErrorReply struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
