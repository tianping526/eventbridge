package data

import (
	"fmt"
	"time"

	kz "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/tianping526/eventbridge/app/service/internal/conf"
)

func NewLogger(ai *conf.AppInfo, cfg *conf.Bootstrap) (log.Logger, func(), error) {
	level := conf.Log_INFO
	encoding := conf.Log_JSON
	sampling := &zap.SamplingConfig{
		Initial:    100,
		Thereafter: 100,
	}
	outputPaths := []*conf.Log_Output{{Path: "stderr"}}

	if cfg.Log != nil {
		level = cfg.Log.Level
		encoding = cfg.Log.Encoding
		if cfg.Log.Sampling != nil {
			sampling = &zap.SamplingConfig{
				Initial:    int(cfg.Log.Sampling.Initial),
				Thereafter: int(cfg.Log.Sampling.Thereafter),
			}
		}
		if len(cfg.Log.OutputPaths) > 0 {
			outputPaths = cfg.Log.OutputPaths
		}
	}

	// encoder
	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = zapcore.OmitKey
	encoderConfig.NameKey = zapcore.OmitKey
	encoderConfig.CallerKey = zapcore.OmitKey
	encoderConfig.StacktraceKey = zapcore.OmitKey
	if encoding == conf.Log_CONSOLE {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// sinks
	var sink zapcore.WriteSyncer
	closes := make([]func(), 0, len(outputPaths))
	paths := make([]string, 0, len(outputPaths))
	syncer := make([]zapcore.WriteSyncer, 0, len(outputPaths))
	for _, out := range outputPaths {
		if out.Rotate == nil {
			paths = append(paths, out.Path)
			continue
		}

		lg := &lumberjack.Logger{
			Filename:   out.Path,
			MaxSize:    int(out.Rotate.MaxSize),
			MaxAge:     int(out.Rotate.MaxAge),
			MaxBackups: int(out.Rotate.MaxBackups),
			Compress:   out.Rotate.Compress,
		}

		syncer = append(syncer, zapcore.AddSync(lg))
		closes = append(closes, func() {
			err := lg.Close()
			if err != nil {
				fmt.Printf("close lumberjack logger(%s) error(%s))", out.Path, err)
			}
		})
	}
	if len(paths) > 0 {
		writer, mc, err := zap.Open(paths...)
		if err != nil {
			for _, c := range closes {
				c()
			}
			return nil, nil, err
		}
		closes = append(closes, mc)
		syncer = append(syncer, writer)
	}
	sink = zap.CombineWriteSyncers(syncer...)

	zl := zap.New(
		zapcore.NewCore(encoder, sink, zap.NewAtomicLevelAt(zapcore.Level(level-1))),
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				sampling.Initial,
				sampling.Thereafter,
			)
		}),
	)

	logger := log.With(
		kz.NewLogger(zl),
		"ts", log.DefaultTimestamp,
		"service.id", ai.Id,
		"service.name", ai.Name,
		"service.version", ai.Version,
		"trace_id", tracing.TraceID(),
		"span_id", tracing.SpanID(),
	)
	return logger, func() {
		err := zl.Sync()
		if err != nil {
			fmt.Printf("sync logger error(%s)", err)
		}
		for _, c := range closes {
			c()
		}
	}, nil
}
