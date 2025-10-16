package logger

import (
	"flag"
	"fmt"

	"k8s.io/klog/v2"
)

func InitFlags(fs *flag.FlagSet) {
	klog.InitFlags(fs)
}

func Info(msg string, keysAndValues ...interface{}) {
	klog.InfoS(msg, keysAndValues...)
}

func InfoS(msg string, keysAndValues ...interface{}) {
	klog.InfoS(msg, keysAndValues...)
}

func V(level int) klog.Verbose {
	return klog.V(klog.Level(level))
}

func Error(err error, msg string, keysAndValues ...interface{}) {
	klog.ErrorS(err, msg, keysAndValues...)
}

func ErrorS(err error, msg string, keysAndValues ...interface{}) {
	klog.ErrorS(err, msg, keysAndValues...)
}

func Warning(msg string, keysAndValues ...interface{}) {
	klog.InfoS("WARNING: "+msg, keysAndValues...)
}

func WarningS(msg string, keysAndValues ...interface{}) {
	klog.InfoS("WARNING: "+msg, keysAndValues...)
}

func Fatal(msg string, keysAndValues ...interface{}) {
	klog.FatalDepth(1, fmt.Sprint(msg, keysAndValues))
}

func FatalS(msg string, keysAndValues ...interface{}) {
	klog.Fatalf(msg, keysAndValues...)
}

func Flush() {
	klog.Flush()
}

type NamedLogger struct {
	name string
}

func WithName(name string) *NamedLogger {
	return &NamedLogger{name: name}
}

func (l *NamedLogger) Info(msg string, keysAndValues ...interface{}) {
	klog.InfoS(fmt.Sprintf("[%s] %s", l.name, msg), keysAndValues...)
}

func (l *NamedLogger) InfoS(msg string, keysAndValues ...interface{}) {
	klog.InfoS(fmt.Sprintf("[%s] %s", l.name, msg), keysAndValues...)
}

func (l *NamedLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	klog.ErrorS(err, fmt.Sprintf("[%s] %s", l.name, msg), keysAndValues...)
}

func (l *NamedLogger) Warning(msg string, keysAndValues ...interface{}) {
	klog.InfoS(fmt.Sprintf("[%s] WARNING: %s", l.name, msg), keysAndValues...)
}

func (l *NamedLogger) V(level int) klog.Verbose {
	return klog.V(klog.Level(level))
}
