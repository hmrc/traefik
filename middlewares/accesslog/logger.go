package accesslog

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/containous/traefik/types"
)

type key string

const (
	// DataTableKey is the key within the request context used to
	// store the Log Data Table
	DataTableKey key = "LogDataTable"
)

// LogHandler will write each request and its response to the access log.
type LogHandler struct {
	logger *logrus.Logger
	file   *os.File
}

// NewLogHandler creates a new LogHandler
func NewLogHandler(config *types.AccessLog) (*LogHandler, error) {
	if len(config.FilePath) == 0 {
		return nil, errors.New("Empty file path specified for accessLogsFile")
	}

	dir := filepath.Dir(config.FilePath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log path %s: %s", dir, err)
	}

	file, err := os.OpenFile(config.FilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %s %s", dir, err)
	}

	var formatter logrus.Formatter

	switch config.Format {
	case "common":
		formatter = new(CommonLogFormatter)
	case "json":
		formatter = new(logrus.JSONFormatter)
	default:
		return nil, fmt.Errorf("unsupported access log format: %s", config.Format)
	}

	logger := &logrus.Logger{
		Out:       file,
		Formatter: formatter,
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}
	return &LogHandler{logger: logger, file: file}, nil
}

// GetLogDataTable gets the request context object that contains logging data. This accretes
// data as the request passes through the middleware chain.
func GetLogDataTable(req *http.Request) *LogData {
	return req.Context().Value(DataTableKey).(*LogData)
}

func (l *LogHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	now := time.Now().UTC()
	core := make(CoreLogData)

	logDataTable := &LogData{Core: core, Request: req.Header}
	core[StartUTC] = now
	core[StartLocal] = now.Local()

	reqWithDataTable := req.WithContext(context.WithValue(req.Context(), DataTableKey, logDataTable))

	var crr *captureRequestReader
	if req.Body != nil {
		crr = &captureRequestReader{source: req.Body, count: 0}
		reqWithDataTable.Body = crr
	}

	core[RequestCount] = nextRequestCount()
	if req.Host != "" {
		core[RequestAddr] = req.Host
		core[RequestHost], core[RequestPort] = silentSplitHostPort(req.Host)
	}
	// copy the URL without the scheme, hostname etc
	urlCopy := &url.URL{
		Path:       req.URL.Path,
		RawPath:    req.URL.RawPath,
		RawQuery:   req.URL.RawQuery,
		ForceQuery: req.URL.ForceQuery,
		Fragment:   req.URL.Fragment,
	}
	urlCopyString := urlCopy.String()
	core[RequestMethod] = req.Method
	core[RequestPath] = urlCopyString
	core[RequestProtocol] = req.Proto
	core[RequestLine] = fmt.Sprintf("%s %s %s", req.Method, urlCopyString, req.Proto)

	core[ClientAddr] = req.RemoteAddr
	core[ClientHost], core[ClientPort] = silentSplitHostPort(req.RemoteAddr)
	core[ClientUsername] = usernameIfPresent(req.URL)

	crw := &captureResponseWriter{rw: rw}

	next.ServeHTTP(crw, reqWithDataTable)

	logDataTable.DownstreamResponse = crw.Header()
	l.logTheRoundTrip(logDataTable, crr, crw)
}

// Close closes the Logger (i.e. the file etc).
func (l *LogHandler) Close() error {
	return l.file.Close()
}

func silentSplitHostPort(value string) (host string, port string) {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return value, "-"
	}
	return host, port
}

func usernameIfPresent(theURL *url.URL) string {
	username := "-"
	if theURL.User != nil {
		if name := theURL.User.Username(); name != "" {
			username = name
		}
	}
	return username
}

// Logging handler to log frontend name, backend name, and elapsed time
func (l *LogHandler) logTheRoundTrip(logDataTable *LogData, crr *captureRequestReader, crw *captureResponseWriter) {

	core := logDataTable.Core

	if crr != nil {
		core[RequestContentSize] = crr.count
	}

	core[DownstreamStatus] = crw.Status()
	core[DownstreamStatusLine] = fmt.Sprintf("%03d %s", crw.Status(), http.StatusText(crw.Status()))
	core[DownstreamContentSize] = crw.Size()
	if original, ok := core[OriginContentSize]; ok {
		o64 := original.(int64)
		if o64 != crw.Size() && 0 != crw.Size() {
			core[GzipRatio] = float64(o64) / float64(crw.Size())
		}
	}

	// n.b. take care to perform time arithmetic using UTC to avoid errors at DST boundaries
	total := time.Now().UTC().Sub(core[StartUTC].(time.Time))
	core[Duration] = total
	if origin, ok := core[OriginDuration]; ok {
		core[Overhead] = total - origin.(time.Duration)
	} else {
		core[Overhead] = total
	}

	fields := logrus.Fields{}

	for k, v := range logDataTable.Core {
		fields[k] = v
	}

	for k := range logDataTable.Request {
		fields["request_"+k] = logDataTable.Request.Get(k)
	}

	for k := range logDataTable.OriginResponse {
		fields["origin_"+k] = logDataTable.OriginResponse.Get(k)
	}

	for k := range logDataTable.DownstreamResponse {
		fields["downstream_"+k] = logDataTable.DownstreamResponse.Get(k)
	}

	l.logger.WithFields(fields).Println()
}

//-------------------------------------------------------------------------------------------------

var requestCounter uint64 // Request ID

func nextRequestCount() uint64 {
	return atomic.AddUint64(&requestCounter, 1)
}
