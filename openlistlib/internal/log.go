package internal

import log "github.com/sirupsen/logrus"

// MyFormatter 是一个 logrus 格式化器，在 Format 中会调用 OnLog 回调，
// 由 gomobile 端接管实际的日志输出。
type MyFormatter struct {
	log.Formatter
	OnLog func(entry *log.Entry)
}

// Format 实现 logrus.Formatter 接口，转发日志条目到 OnLog 回调。
func (f *MyFormatter) Format(entry *log.Entry) ([]byte, error) {
	f.OnLog(entry)
	return nil, nil
}
