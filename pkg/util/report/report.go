package report

import (
	"context"
	"strings"
	"sync"

	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// errors
var (
	ErrNoReport = errors.New("no report")
)

type contextKey struct{}

type Error struct {
	Token   string `json:"token"`
	Cause   string `json:"cause"`
	Message string `json:"msg"`
	err     error
}

// Entry represents a single Log entry of the report
type Entry struct {
	Timestamp timestamp.Timestamp `json:"timestamp"`
	Level     zapcore.Level       `json:"lvl"`
	Message   string              `json:"msg"`
	Fields    []zap.Field         `json:"fields"`
}

// Log is a named slice used inside the report
type Log []Entry

func (l *Log) AddEntry(lvl zapcore.Level, msg string, fields ...zap.Field) {
	if l == nil {
		*l = Log{}
	}

	*l = append(*l, Entry{
		Timestamp: timestamp.Now(),
		Level:     lvl,
		Message:   msg,
		Fields:    fields,
	})
}

type Report struct {
	Err    Error `json:"error"`
	Log    Log   `json:"log"`
	logger *zap.Logger
	sync.RWMutex
}

func New(l *zap.Logger) *Report {
	return &Report{
		logger: l,
	}
}

func NewWithContext(parent context.Context, l *zap.Logger) (*Report, context.Context) {
	rep := New(l)
	return rep, context.WithValue(parent, contextKey{}, rep)
}

func FromContext(ctx context.Context) (rep *Report, err error) {
	rep, ok := ctx.Value(contextKey{}).(*Report)

	if !ok || rep == nil {
		return nil, ErrNoReport
	}

	return rep, nil
}

func (rep *Report) HasError() bool {
	rep.RLock()
	hasError := rep.Err.err != nil
	rep.RUnlock()

	return hasError
}

func (rep *Report) WithError(token string, err error) *Report {
	// doing nothing if error is nil
	if err == nil {
		return rep
	}

	rep.Err = Error{
		Token:   strings.ToLower(token),
		Cause:   errors.Cause(err).Error(),
		Message: err.Error(),
		err:     err,
	}

	return rep
}

func (rep *Report) Wrap(token string, err error, msg string) *Report {
	rep.Lock()
	rep.Err = Error{
		Token:   strings.ToLower(token),
		Cause:   errors.Cause(err).Error(),
		Message: err.Error(),
		err:     errors.Wrap(err, msg),
	}
	rep.Unlock()

	return rep
}

func (rep *Report) Wrapf(token string, err error, msg string, args ...interface{}) *Report {
	rep.Lock()
	rep.Err = Error{
		Token:   strings.ToLower(token),
		Cause:   err.Error(),
		Message: msg,
		err:     errors.Wrapf(err, msg, args...),
	}
	rep.Unlock()

	return rep
}

func (rep *Report) Debug(msg string, fields ...zap.Field) {
	if rep.logger != nil {
		rep.logger.Debug(msg, fields...)
	}

	rep.Log.AddEntry(zap.DebugLevel, msg, fields...)
}

func (rep *Report) Info(msg string, fields ...zap.Field) {
	if rep.logger != nil {
		rep.logger.Info(msg, fields...)
	}

	rep.Log.AddEntry(zap.InfoLevel, msg, fields...)
}

func (rep *Report) Warn(msg string, fields ...zap.Field) {
	if rep.logger != nil {
		rep.logger.Warn(msg, fields...)
	}

	rep.Log.AddEntry(zap.WarnLevel, msg, fields...)
}

func (rep *Report) Error(msg string, fields ...zap.Field) {
	if rep.logger != nil {
		rep.logger.Error(msg, fields...)
	}

	rep.Log.AddEntry(zap.ErrorLevel, msg, fields...)
}
