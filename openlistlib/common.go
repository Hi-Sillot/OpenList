package openlistlib

import "net"

// GetOutboundIP 通过向 8.8.8.8 发送 UDP 包获取本机出口 IP。
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// GetOutboundIPString 返回本机出口 IP 的字符串形式，失败时返回 "localhost"。
func GetOutboundIPString() string {
	netIp, err := GetOutboundIP()
	if err != nil {
		return "localhost"
	}

	return netIp.String()
}
