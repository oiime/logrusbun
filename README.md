# logrusbun

A simple hook for bun that enables logging with logrus


    go get github.com/oiime/logrusbun


## Usage

```golang
db := bun.NewDB(...)
log := logrus.New()
db.AddQueryHook(NewQueryHook(QueryHookOptions{Logger: log}))

```

### QueryHookOptions

* _LogSlow_ time.Duration value of queries considered 'slow'
* _Logger_ logger following logrus.FieldLogger interface
* _QueryLevel_ logrus.Level for logging queries, eg: QueryLevel: logrus.DebugLevel
* _SlowLevel_ logrus.Level for logging slow queries
* _ErrorLevel_ logrus.Level for logging errors
* _MessageTemplate_ alternative message string template, avialable variables listed below
* _ErrorTemplate_ alternative error string template, available variables listed below

### Message template variables

* {{.Timestamp}} Event timestmap
* {{.Duration}} Duration of query
* {{.Query}} Query string
* {{.Operation}} Operation name (eg: SELECT, UPDATE...)
* {{.Error}} Error message if available

### Kitchen sink example
```golang
db.AddQueryHook(NewQueryHook(QueryHookOptions{
    LogSlow:    time.Second,
    Logger:     log,
    QueryLevel: logrus.DebugLevel,
    ErrorLevel: logrus.ErrorLevel,
    SlowLevel:  logrus.WarnLevel,
    MessageTemplate: "{{.Operation}}[{{.Duration}}]: {{.Query}}",
    ErrorTemplate: "{{.Operation}}[{{.Duration}}]: {{.Query}}: {{.Error}",
}))

```
