package log

import (
	"context"
	"dev.hackerman.me/artheon/veverse-shared/model"
	"github.com/sirupsen/logrus"
)

type ClickhouseSyslogLogrusHook struct {
	ctx    context.Context
	levels []logrus.Level
}

func NewHook(ctx context.Context) (*ClickhouseSyslogLogrusHook, error) {
	hook := &ClickhouseSyslogLogrusHook{
		ctx:    ctx,
		levels: nil,
	}

	return hook, nil
}

func (hook *ClickhouseSyslogLogrusHook) Levels() []logrus.Level {
	if hook.levels == nil {
		return []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		}
	}

	return hook.levels
}

func (hook *ClickhouseSyslogLogrusHook) Fire(entry *logrus.Entry) error {
	var metadata model.SystemLogRequest
	metadata.Service = "pixel-streaming-launcher"
	metadata.Timestamp = entry.Time
	metadata.Level = entry.Level.String()
	metadata.Message = entry.Message
	metadata.Payload = entry.Data
	return model.ReportSystemLog(hook.ctx, metadata)
}
