package qdrant

import (
	"context"
	"log/slog"

	"github.com/yaoapp/kun/log"
)

func init() {
	// Set the default slog handler to bridge to kun/log
	// This ensures qdrant client logs are unified with gou's logging system
	slog.SetDefault(slog.New(&kunLogHandler{}))
}

// kunLogHandler bridges slog to kun/log
type kunLogHandler struct {
	attrs  []slog.Attr
	groups []string
}

// Enabled implements slog.Handler
func (h *kunLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	kunLevel := log.GetLevel()
	switch level {
	case slog.LevelDebug:
		return kunLevel >= log.DebugLevel
	case slog.LevelInfo:
		return kunLevel >= log.InfoLevel
	case slog.LevelWarn:
		return kunLevel >= log.WarnLevel
	case slog.LevelError:
		return kunLevel >= log.ErrorLevel
	default:
		return true
	}
}

// Handle implements slog.Handler
func (h *kunLogHandler) Handle(_ context.Context, r slog.Record) error {
	// Build fields from attributes
	fields := log.F{}
	for _, attr := range h.attrs {
		fields[attr.Key] = attr.Value.Any()
	}
	r.Attrs(func(attr slog.Attr) bool {
		fields[attr.Key] = attr.Value.Any()
		return true
	})

	msg := r.Message
	var entry *log.Entry
	if len(fields) > 0 {
		entry = log.With(fields)
	}

	switch r.Level {
	case slog.LevelDebug:
		if entry != nil {
			entry.Debug("%s", msg)
		} else {
			log.Debug("%s", msg)
		}
	case slog.LevelInfo:
		if entry != nil {
			entry.Info("%s", msg)
		} else {
			log.Info("%s", msg)
		}
	case slog.LevelWarn:
		if entry != nil {
			entry.Warn("%s", msg)
		} else {
			log.Warn("%s", msg)
		}
	case slog.LevelError:
		if entry != nil {
			entry.Error("%s", msg)
		} else {
			log.Error("%s", msg)
		}
	default:
		if entry != nil {
			entry.Info("%s", msg)
		} else {
			log.Info("%s", msg)
		}
	}

	return nil
}

// WithAttrs implements slog.Handler
func (h *kunLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &kunLogHandler{
		attrs:  make([]slog.Attr, len(h.attrs)+len(attrs)),
		groups: h.groups,
	}
	copy(newHandler.attrs, h.attrs)
	copy(newHandler.attrs[len(h.attrs):], attrs)
	return newHandler
}

// WithGroup implements slog.Handler
func (h *kunLogHandler) WithGroup(name string) slog.Handler {
	newHandler := &kunLogHandler{
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
	return newHandler
}
