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

// LogCallback 用于把 logrus 输出桥接到 gomobile 端（Android Logcat / iOS os_log）。
// Init 阶段会注册到 utils.Log.SetFormatter，见 openlistlib/server.go::Init。
//
// 字段类型约束（gobind 自动生成 Java 接口的映射规则）：
//   - level: int16  → Java short  （logrus.Level 范围 [-7, 7]，int16 足够）
//   - time:  int64  → Java long   （entry.Time.UnixMilli()，毫秒时间戳）
//   - log:   string → Java String （entry.Message 文本）
//
// 若改字段类型或顺序，OpenListBridge.kt 的 onLog() 实现必须同步更新。
type LogCallback interface {
	OnLog(level int16, time int64, log string)
}
