package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"
)

type PrettyHandler struct {
	level  slog.Level
	output *os.File
	attrs  []slog.Attr
	groups []string
}

func NewPrettyHandler(output *os.File, level slog.Level) *PrettyHandler {
	return &PrettyHandler{
		level:  level,
		output: output,
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	// Format level
	levelStr := r.Level.String()
	switch r.Level {
	case slog.LevelDebug:
		levelStr = "\033[35mDBG\033[0m"
	case slog.LevelInfo:
		levelStr = "\033[32mINF\033[0m"
	case slog.LevelWarn:
		levelStr = "\033[33mWRN\033[0m"
	case slog.LevelError:
		levelStr = "\033[31mERR\033[0m"
	}

	// Collect all attributes
	attrs := make(map[string]any)
	for _, a := range h.attrs {
		attrs[a.Key] = a.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	// Format time and message
	timeStr := r.Time.Format(time.DateTime)
	msg := "\033[94m" + r.Message + "\033[0m"

	// Print main line
	_, _ = h.output.WriteString(timeStr + " " + levelStr + " " + msg + "\n")

	// Print attributes as formatted JSON if any
	if len(attrs) > 0 {
		jsonBytes, err := json.MarshalIndent(attrs, "", "  ")
		if err == nil {
			_, _ = h.output.WriteString("\033[90m" + string(jsonBytes) + "\033[0m\n")
		}
	}

	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &PrettyHandler{
		level:  h.level,
		output: h.output,
		attrs:  make([]slog.Attr, len(h.attrs)+len(attrs)),
		groups: h.groups,
	}
	copy(newHandler.attrs, h.attrs)
	copy(newHandler.attrs[len(h.attrs):], attrs)
	return newHandler
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{
		level:  h.level,
		output: h.output,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

