package logrusbun

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
)

type Option func(hook *QueryHook)

// WithEnabled enables/disables this hook
func WithEnabled(on bool) Option {
	return func(h *QueryHook) {
		h.enabled = on
	}
}

// WithVerbose configures the hook to log all queries
// (by default, only failed queries are logged)
func WithVerbose(on bool) Option {
	return func(h *QueryHook) {
		h.verbose = on
	}
}

// FromEnv configures the hook using the environment variable value.
// For example, WithEnv("BUNDEBUG"):
//   - BUNDEBUG=0 - disables the hook.
//   - BUNDEBUG=1 - enables the hook.
//   - BUNDEBUG=2 - enables the hook and verbose mode.
func FromEnv(keys ...string) Option {
	if len(keys) == 0 {
		keys = []string{"BUNDEBUG"}
	}
	return func(h *QueryHook) {
		for _, key := range keys {
			if env, ok := os.LookupEnv(key); ok {
				h.enabled = env != "" && env != "0"
				h.verbose = env == "2"
				break
			}
		}
	}
}

// WithQueryHookOptions allows setting the initial logging options
// for logrus
func WithQueryHookOptions(opts QueryHookOptions) Option {
	return func(h *QueryHook) {
		if opts.ErrorTemplate == "" {
			opts.ErrorTemplate = "{{.Operation}}[{{.Duration}}]: {{.Query}}: {{.Error}}"
		}
		if opts.MessageTemplate == "" {
			opts.MessageTemplate = "{{.Operation}}[{{.Duration}}]: {{.Query}}"
		}
		h.opts = &opts
		errorTemplate, err := template.New("ErrorTemplate").Parse(h.opts.ErrorTemplate)
		if err != nil {
			panic(err)
		}
		messageTemplate, err := template.New("MessageTemplate").Parse(h.opts.MessageTemplate)
		if err != nil {
			panic(err)
		}

		h.errorTemplate = errorTemplate
		h.messageTemplate = messageTemplate
		h.opts = &opts
	}
}

// QueryHookOptions logging options
type QueryHookOptions struct {
	LogSlow         time.Duration
	Logger          logrus.FieldLogger
	QueryLevel      logrus.Level
	SlowLevel       logrus.Level
	ErrorLevel      logrus.Level
	MessageTemplate string
	ErrorTemplate   string
}

// QueryHook wraps query hook
type QueryHook struct {
	enabled         bool
	verbose         bool
	opts            *QueryHookOptions
	errorTemplate   *template.Template
	messageTemplate *template.Template
}

// LogEntryVars variables made available t otemplate
type LogEntryVars struct {
	Timestamp time.Time
	Query     string
	Operation string
	Duration  time.Duration
	Error     error
}

// NewQueryHook returns new instance
func NewQueryHook(options ...Option) *QueryHook {
	h := new(QueryHook)

	for _, opt := range options {
		opt(h)
	}

	if h.opts == nil {
		panic("logrus settings not set.")
	}

	return h
}

// BeforeQuery does nothing tbh
func (h *QueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery convert a bun QueryEvent into a logrus message
func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	if !h.enabled {
		return
	}

	if !h.verbose {
		switch event.Err {
		case nil, sql.ErrNoRows, sql.ErrTxDone:
			return
		}
	}
	var level logrus.Level
	var isError bool
	var msg bytes.Buffer

	now := time.Now()
	dur := now.Sub(event.StartTime)

	switch event.Err {
	case nil, sql.ErrNoRows:
		isError = false
		if h.opts.LogSlow > 0 && dur >= h.opts.LogSlow {
			level = h.opts.SlowLevel
		} else {
			level = h.opts.QueryLevel
		}
	default:
		isError = true
		level = h.opts.ErrorLevel
	}
	if level == 0 {
		return
	}

	args := &LogEntryVars{
		Timestamp: now,
		Query:     string(event.Query),
		Operation: eventOperation(event),
		Duration:  dur,
		Error:     event.Err,
	}

	if isError {
		if err := h.errorTemplate.Execute(&msg, args); err != nil {
			panic(err)
		}
	} else {
		if err := h.messageTemplate.Execute(&msg, args); err != nil {
			panic(err)
		}
	}

	switch level {
	case logrus.DebugLevel:
		h.opts.Logger.Debug(msg.String())
	case logrus.InfoLevel:
		h.opts.Logger.Info(msg.String())
	case logrus.WarnLevel:
		h.opts.Logger.Warn(msg.String())
	case logrus.ErrorLevel:
		h.opts.Logger.Error(msg.String())
	case logrus.FatalLevel:
		h.opts.Logger.Fatal(msg.String())
	case logrus.PanicLevel:
		h.opts.Logger.Panic(msg.String())
	default:
		panic(fmt.Errorf("Unsupported level: %v", level))
	}

}

// taken from bun
func eventOperation(event *bun.QueryEvent) string {
	switch event.QueryAppender.(type) {
	case *bun.SelectQuery:
		return "SELECT"
	case *bun.InsertQuery:
		return "INSERT"
	case *bun.UpdateQuery:
		return "UPDATE"
	case *bun.DeleteQuery:
		return "DELETE"
	case *bun.CreateTableQuery:
		return "CREATE TABLE"
	case *bun.DropTableQuery:
		return "DROP TABLE"
	}
	return queryOperation(event.Query)
}

// taken from bun
func queryOperation(name string) string {
	if idx := strings.Index(name, " "); idx > 0 {
		name = name[:idx]
	}
	if len(name) > 16 {
		name = name[:16]
	}
	return string(name)
}
