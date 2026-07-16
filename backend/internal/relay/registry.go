package relay

import (
	"sync"
)

// Target 描述前端希望后端连往的远端地址。
// 由 HTTP 层解析查询参数后传入，协议处理器据此建立后端连接。
//   - tcp/udp/mqtt：使用 Host/Port 建立 socket。
//   - ws/socketio：使用 URL 建立 WebSocket 客户端，Headers 携带自定义请求头，
//     Insecure 用于跳过 TLS 校验（访问内网/自签目标）。
type Target struct {
	Host        string
	Port        string
	URL         string
	Headers     map[string]string
	Subprotocol string
	Insecure    bool
}

// ProtocolHandler 是后端透传类协议（UDP/TCP/WS/Socket.IO/MQTT）的统一扩展点。
// 新增协议只需实现该接口并 Register，核心路由无需改动。
// 注意：GraphQL/MCP/AI 目前不是 relay 协议（未注册对应 Handler），gRPC 走独立的 /ws/grpc 通道。
type ProtocolHandler interface {
	// Name 返回协议标识，如 "udp" / "tcp" / "ws" / "socketio" / "mqtt"。
	Name() string
	// Serve 在一个已升级的 WS 连接上，建立到 target 的 socket 并双向透传数据。
	Serve(conn Conn, target Target)
}

// Conn 抽象 WS 连接，解耦具体 WS 库实现。
// 区分消息类型是为了让控制帧（JSON 文本）与二进制数据帧走不同通道。
type Conn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
}

var (
	mu       sync.RWMutex
	registry = map[string]ProtocolHandler{}
)

// Register 注册一个协议处理器，重复注册后者覆盖。
func Register(h ProtocolHandler) {
	mu.Lock()
	defer mu.Unlock()
	registry[h.Name()] = h
}

// Get 取出已注册的协议处理器。
func Get(name string) (ProtocolHandler, bool) {
	mu.RLock()
	defer mu.RUnlock()
	h, ok := registry[name]
	return h, ok
}
