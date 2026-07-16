package relay

import (
	"net"
	"time"
)

// TCPHandler 通过后端建立一条 TCP 长连接，与前端 WS 双向透传字节流。
type TCPHandler struct{}

func (TCPHandler) Name() string { return "tcp" }

func (TCPHandler) Serve(c Conn, target Target) {
	addr := net.JoinHostPort(target.Host, target.Port)
	remote, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		writeCtrl(c, CtrlError, err.Error())
		return
	}
	defer remote.Close()
	defer c.Close()

	writeCtrl(c, CtrlConnected, "")
	writeStatus(c, remote.LocalAddr().String(), remote.RemoteAddr().String())

	relayStream(c, remote, false)
}
