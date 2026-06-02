package openlistlib

// Event 用于在 gomobile 端接收来自 Go 库的服务端事件回调。
type Event interface {
	// OnStartError 服务启动失败时回调，t 标识协议（http/https/unix）。
	OnStartError(t string, err string)
	// OnShutdown 服务正常关闭后回调。
	OnShutdown(t string)
	// OnProcessExit 当 utils.Log.Fatal 触发进程退出前回调，code 为退出码。
	OnProcessExit(code int)
}
