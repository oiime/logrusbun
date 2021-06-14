package logrusbun

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
)

func TestLogging(t *testing.T) {
	var log = &logrus.Logger{
		Formatter: &testFormatter{
			cb: func(*logrus.Entry) ([]byte, error) {
				return nil, nil
			},
		},
		Level: logrus.DebugLevel,
	}
	db := bun.DB{}
	db.AddQueryHook(NewQueryHook(QueryHookOptions{Logger: log}))
	// @TODO against empty db (stub?)
}

type testFormatter struct {
	cb func(*logrus.Entry) ([]byte, error)
}

func (f *testFormatter) Format(e *logrus.Entry) ([]byte, error) {
	return f.cb(e)
}
