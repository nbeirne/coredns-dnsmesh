package browser


// Logger defines a standard logging interface that can be implemented by
// various logging libraries, such as the CoreDNS logger or the standard log package.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type NoLogger struct {
}

func (_ NoLogger)Debugf(format string, args...interface{}) {}
func (_ NoLogger)Infof(format string, args...interface{}) {}
func (_ NoLogger)Warningf(format string, args...interface{}) {}
func (_ NoLogger)Errorf(format string, args...interface{}) {}

