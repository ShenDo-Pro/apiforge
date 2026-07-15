package grpcproxy

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// 前端 -> 后端 的 WS 文本消息
type wsMsg struct {
	Type    string          `json:"type"` // connect / list / invoke
	Target  string          `json:"target"`
	Service string          `json:"service"`
	Method  string          `json:"method"`
	Data    json.RawMessage `json:"data"`
}

type methodInfo struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

type svcInfo struct {
	Name    string       `json:"name"`
	Methods []methodInfo `json:"methods"`
}

// 后端 -> 前端的 WS 文本消息
type wsResp struct {
	Type     string    `json:"type"` // connected / list / result / error
	Message  string    `json:"message,omitempty"`
	Services []svcInfo `json:"services,omitempty"`
	Service  string    `json:"service,omitempty"`
	Method   string    `json:"method,omitempty"`
	Data     any       `json:"data,omitempty"`
}

func writeJSON(c *websocket.Conn, v any) {
	b, _ := json.Marshal(v)
	_ = c.WriteMessage(websocket.TextMessage, b)
}

func writeErr(c *websocket.Conn, msg string) {
	writeJSON(c, wsResp{Type: "error", Message: msg})
}

// Handler 是 /ws/grpc 端点：基于服务端反射把任意 gRPC 服务暴露给前端调试。
// 协议流程：前端发 connect(target) -> 后端 dial 并校验反射；list 返回服务/方法树；
// invoke(service,method,data) 用 dynamicpb 构造请求并调用，protojson 转 JSON 回传。
func Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var gc *grpc.ClientConn
	var rc *grpcreflect.Client
	ctx := r.Context()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var m wsMsg
		if err := json.Unmarshal(data, &m); err != nil {
			writeErr(conn, "invalid json")
			continue
		}
		switch m.Type {
		case "connect":
			if gc != nil {
				_ = gc.Close()
			}
			g, e := grpc.NewClient(m.Target, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if e != nil {
				writeErr(conn, e.Error())
				continue
			}
			rc = grpcreflect.NewClientAuto(ctx, g)
			if _, e := rc.ListServices(); e != nil {
				_ = g.Close()
				gc = nil
				writeErr(conn, "reflection failed: "+e.Error())
				continue
			}
			gc = g
			writeJSON(conn, wsResp{Type: "connected", Message: m.Target})

		case "list":
			if rc == nil {
				writeErr(conn, "not connected")
				continue
			}
			names, e := rc.ListServices()
			if e != nil {
				writeErr(conn, e.Error())
				continue
			}
			svcs := make([]svcInfo, 0, len(names))
			for _, n := range names {
				if n == "grpc.reflection.v1.ServerReflection" || n == "grpc.reflection.v1alpha.ServerReflection" {
					continue
				}
				sd, e := rc.ResolveService(n)
				if e != nil {
					continue
				}
				ms := make([]methodInfo, 0, len(sd.GetMethods()))
				for _, md := range sd.GetMethods() {
					ms = append(ms, methodInfo{
						Name:   md.GetName(),
						Input:  md.GetInputType().GetFullyQualifiedName(),
						Output: md.GetOutputType().GetFullyQualifiedName(),
					})
				}
				svcs = append(svcs, svcInfo{Name: n, Methods: ms})
			}
			writeJSON(conn, wsResp{Type: "list", Services: svcs})

		case "invoke":
			if rc == nil || gc == nil {
				writeErr(conn, "not connected")
				continue
			}
			sd, e := rc.ResolveService(m.Service)
			if e != nil {
				writeErr(conn, e.Error())
				continue
			}
			md := sd.FindMethodByName(m.Method)
			if md == nil {
				writeErr(conn, "method not found: "+m.Method)
				continue
			}
			reqMsg := dynamic.NewMessage(md.GetInputType())
			if len(m.Data) > 0 {
				if e := reqMsg.UnmarshalJSON(m.Data); e != nil {
					writeErr(conn, "invalid request json: "+e.Error())
					continue
				}
			}
			respMsg := dynamic.NewMessage(md.GetOutputType())
			full := "/" + m.Service + "/" + m.Method
			ictx, cancel := context.WithTimeout(ctx, 30*time.Second)
			e = gc.Invoke(ictx, full, reqMsg, respMsg)
			cancel()
			if e != nil {
				writeErr(conn, e.Error())
				continue
			}
			out, e := respMsg.MarshalJSON()
			if e != nil {
				writeErr(conn, e.Error())
				continue
			}
			var data any
			_ = json.Unmarshal(out, &data)
			writeJSON(conn, wsResp{Type: "result", Service: m.Service, Method: m.Method, Data: data})

		default:
			writeErr(conn, "unknown type: "+m.Type)
		}
	}
}
