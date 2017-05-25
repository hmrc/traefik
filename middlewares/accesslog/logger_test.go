package accesslog

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/containous/traefik/types"
	shellwords "github.com/mattn/go-shellwords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type logtestResponseWriter struct{}

var (
	logger            *LogHandler
	logfileNameSuffix       = "/traefik/logger/test.log"
	helloWorld              = "Hello, World"
	testBackendName         = "http://127.0.0.1/testBackend"
	testFrontendName        = "testFrontend"
	testStatus              = 123
	testContentSize   int64 = 12
	testHostname            = "TestHost"
	testUsername            = "TestUser"
	testPath                = "testpath"
	testPort                = 8181
	testProto               = "HTTP/0.0"
	testMethod              = "POST"
	testReferer             = "testReferer"
	testUserAgent           = "testUserAgent"
)

func TestLoggerCLF(t *testing.T) {
	tmpDir, logfilePath := doLogging(t, "common")
	defer os.RemoveAll(tmpDir)

	if logdata, err := ioutil.ReadFile(logfilePath); err != nil {
		fmt.Printf("%s\n%s\n", string(logdata), err.Error())
		assert.Nil(t, err)
	} else if tokens, err := shellwords.Parse(string(logdata)); err != nil {
		fmt.Printf("%s\n", err.Error())
		assert.Nil(t, err)
	} else if assert.Equal(t, 14, len(tokens), printLogdata(logdata)) {
		assert.Equal(t, testHostname, tokens[0], printLogdata(logdata))
		assert.Equal(t, testUsername, tokens[2], printLogdata(logdata))
		assert.Equal(t, fmt.Sprintf("%s %s %s", testMethod, testPath, testProto), tokens[5], printLogdata(logdata))
		assert.Equal(t, fmt.Sprintf("%d", testStatus), tokens[6], printLogdata(logdata))
		assert.Equal(t, fmt.Sprintf("%d", len(helloWorld)), tokens[7], printLogdata(logdata))
		assert.Equal(t, testReferer, tokens[8], printLogdata(logdata))
		assert.Equal(t, testUserAgent, tokens[9], printLogdata(logdata))
		assert.Equal(t, "1", tokens[10], printLogdata(logdata))
		assert.Equal(t, testFrontendName, tokens[11], printLogdata(logdata))
		assert.Equal(t, testBackendName, tokens[12], printLogdata(logdata))
	}
}

func TestLoggerJSON(t *testing.T) {
	tmpDir, logfilePath := doLogging(t, "json")
	defer os.RemoveAll(tmpDir)

	var jsonDataIf interface{}
	if logdata, err := ioutil.ReadFile(logfilePath); err != nil {
		fmt.Printf("%s\n%s\n", string(logdata), err.Error())
		assert.Nil(t, err)
	} else if err := json.Unmarshal(logdata, &jsonDataIf); err != nil {
		fmt.Printf("%s\n", err.Error())
		assert.Nil(t, err)
	}

	var jsonData = jsonDataIf.(map[string]interface{})
	if assert.Equal(t, 28, len(jsonData), printLogdataJSON(jsonData)) {
		assert.Equal(t, testHostname, jsonData[RequestHost], printLogdataJSON(jsonData))
		assert.Equal(t, testHostname, jsonData[RequestAddr], printLogdataJSON(jsonData))
		assert.Equal(t, testMethod, jsonData[RequestMethod], printLogdataJSON(jsonData))
		assert.Equal(t, testPath, jsonData[RequestPath], printLogdataJSON(jsonData))
		assert.Equal(t, testProto, jsonData[RequestProtocol], printLogdataJSON(jsonData))
		assert.Equal(t, "-", jsonData[RequestPort], printLogdataJSON(jsonData))
		assert.Equal(t, fmt.Sprintf("%s %s %s", testMethod, testPath, testProto), jsonData[RequestLine], printLogdataJSON(jsonData))
		assert.Equal(t, float64(testStatus), jsonData[DownstreamStatus], printLogdataJSON(jsonData))
		assert.Equal(t, fmt.Sprintf("%d ", testStatus), jsonData[DownstreamStatusLine], printLogdataJSON(jsonData))
		assert.Equal(t, float64(len(helloWorld)), jsonData[DownstreamContentSize], printLogdataJSON(jsonData))
		assert.Equal(t, float64(len(helloWorld)), jsonData[OriginContentSize], printLogdataJSON(jsonData))
		assert.Equal(t, float64(testStatus), jsonData[OriginStatus], printLogdataJSON(jsonData))
		assert.Equal(t, testReferer, jsonData["request_Referer"], printLogdataJSON(jsonData))
		assert.Equal(t, testUserAgent, jsonData["request_User-Agent"], printLogdataJSON(jsonData))
		assert.Equal(t, testFrontendName, jsonData[FrontendName], printLogdataJSON(jsonData))
		assert.Equal(t, testBackendName, jsonData[BackendURL], printLogdataJSON(jsonData))
		assert.Equal(t, testUsername, jsonData[ClientUsername], printLogdataJSON(jsonData))
		assert.Equal(t, testHostname, jsonData[ClientHost], printLogdataJSON(jsonData))
		assert.Equal(t, fmt.Sprintf("%d", testPort), jsonData[ClientPort], printLogdataJSON(jsonData))
		assert.Equal(t, fmt.Sprintf("%s:%d", testHostname, testPort), jsonData[ClientAddr], printLogdataJSON(jsonData))
		assert.Equal(t, "info", jsonData["level"], printLogdataJSON(jsonData))
		assert.Equal(t, "", jsonData["msg"], printLogdataJSON(jsonData))
		assert.True(t, jsonData[RequestCount].(float64) > 0, printLogdataJSON(jsonData))
		assert.True(t, jsonData[Duration].(float64) > 0, printLogdataJSON(jsonData))
		assert.True(t, jsonData[Overhead].(float64) > 0, printLogdataJSON(jsonData))
		assert.True(t, len(jsonData["time"].(string)) > 0, printLogdataJSON(jsonData))
		assert.True(t, len(jsonData["StartLocal"].(string)) > 0, printLogdataJSON(jsonData))
		assert.True(t, len(jsonData["StartUTC"].(string)) > 0, printLogdataJSON(jsonData))
	}
}

func doLogging(t *testing.T, format string) (string, string) {
	tmp, err := ioutil.TempDir("", format)
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err)
	}
	logfilePath := filepath.Join(tmp, logfileNameSuffix)
	config := types.AccessLog{FilePath: logfilePath, Format: format}
	logger, err = NewLogHandler(&config)
	require.NoError(t, err)
	defer logger.Close()
	if _, err := os.Stat(logfilePath); os.IsNotExist(err) {
		t.Fatalf("logger should create %s", logfilePath)
	}
	req := &http.Request{
		Header: map[string][]string{
			"User-Agent": {testUserAgent},
			"Referer":    {testReferer},
		},
		Proto:      testProto,
		Host:       testHostname,
		Method:     testMethod,
		RemoteAddr: fmt.Sprintf("%s:%d", testHostname, testPort),
		URL: &url.URL{
			User: url.UserPassword(testUsername, ""),
			Path: testPath,
		},
	}
	logger.ServeHTTP(&logtestResponseWriter{}, req, LogWriterTestHandlerFunc)
	return tmp, logfilePath
}

func printLogdata(logdata []byte) string {
	return fmt.Sprintf(
		"\nExpected: %s\n"+
			"Actual:   %s",
		"TestHost - TestUser [13/Apr/2016:07:14:19 -0700] \"POST testpath HTTP/0.0\" 123 12 \"testReferer\" \"testUserAgent\" 1 \"testFrontend\" \"http://127.0.0.1/testBackend\" 1ms",
		string(logdata))
}

func printLogdataJSON(jsonData map[string]interface{}) string {
	return fmt.Sprintf(
		"\nExpected: %s\n"+
			"Actual:   %v",
		"map[ClientAddr:TestHost:8181 Duration:50771 FrontendName:testFrontend RequestLine:POST testpath HTTP/0.0 request_Referer:testReferer request_User-Agent:testUserAgent RequestCount:2 RequestHost:TestHost StartLocal:2017-05-25T11:27:28.797663791+01:00 BackendURL:http://127.0.0.1/testBackend ClientHost:TestHost DownstreamStatusLine:123  Overhead:50771 RequestAddr:TestHost DownstreamStatus:123 OriginStatus:123 RequestPort:- RequestProtocol:HTTP/0.0 ClientPort:8181 OriginContentSize:12 RequestMethod:POST ClientUsername:TestUser RequestPath:testpath level:info msg: time:2017-05-25T11:27:28+01:00 DownstreamContentSize:12 StartUTC:2017-05-25T10:27:28.797663791Z]",
		jsonData)

}

func LogWriterTestHandlerFunc(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte(helloWorld))
	rw.WriteHeader(testStatus)

	logDataTable := GetLogDataTable(r)
	logDataTable.Core[FrontendName] = testFrontendName
	logDataTable.Core[BackendURL] = testBackendName
	logDataTable.Core[OriginStatus] = testStatus
	logDataTable.Core[OriginContentSize] = testContentSize
}

func (lrw *logtestResponseWriter) Header() http.Header {
	return map[string][]string{}
}

func (lrw *logtestResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (lrw *logtestResponseWriter) WriteHeader(s int) {
}
