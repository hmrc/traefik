package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/containous/traefik/integration/try"
	"github.com/go-check/check"
	checker "github.com/vdemeester/shakers"
)

// Log rotation integration test suite
type LogRotationSuite struct{ BaseSuite }

func (s *LogRotationSuite) TestAccessLogRotation(c *check.C) {
	// Start Traefik
	cmd := exec.Command(traefikBinary, "--configFile=fixtures/access_log_config.toml")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()
	defer os.Remove("access.log")
	defer os.Remove("traefik.log")

	// Verify Traefik started OK
	verifyEmptyErrorLog(c, "traefik.log")

	// Start test servers
	ts1 := startAccessLogServer(8081)
	defer ts1.Close()
	ts2 := startAccessLogServer(8082)
	defer ts2.Close()
	ts3 := startAccessLogServer(8083)
	defer ts3.Close()

	// Allow time to startup
	time.Sleep(1 * time.Second)

	// Make some requests
	_, err = http.Get("http://127.0.0.1:8000/test1")
	c.Assert(err, checker.IsNil)
	// in the midst of the requests, issue SIGUSR1 signal to server process and move access log
	err = cmd.Process.Signal(syscall.SIGUSR1)
	c.Assert(err, checker.IsNil)
	os.Rename("access.log", "access.log.rotated")

	//continue issuing requests
	_, err = http.Get("http://127.0.0.1:8000/test2")
	c.Assert(err, checker.IsNil)
	_, err = http.Get("http://127.0.0.1:8000/test2")
	c.Assert(err, checker.IsNil)

	// Verify access.log.rotated output as expected
	rotated, err := os.Open("access.log.rotated")
	c.Assert(err, checker.IsNil)
	rotatedLog := bufio.NewScanner(rotated)
	count := 0
	for rotatedLog.Scan() {
		line := rotatedLog.Text()
		c.Log("rl: " + line)
		if len(line) > 0 {
			CheckAccessLogFormat(c, line, count)
		}
		count++
	}
	c.Assert(count, checker.Equals, 1)

	// Verify access.log output as expected
	file, err := os.Open("access.log")
	c.Assert(err, checker.IsNil)
	accessLog := bufio.NewScanner(file)
	for accessLog.Scan() {
		line := accessLog.Text()
		c.Log("al: " + line)
		if len(line) > 0 {
			CheckAccessLogFormat(c, line, count)
		}
		count++
	}
	// combined, we have 3 lines; i.e. 1 in the initial access.log
	c.Assert(count, checker.Equals, 3)

	verifyEmptyErrorLog(c, "traefik.log")
}

func (s *LogRotationSuite) TestTraefikLogRotation(c *check.C) {
	// Start Traefik
	cmd := exec.Command(traefikBinary, "--configFile=fixtures/traefik_log_config.toml")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()
	defer os.Remove("access.log")
	defer os.Remove("traefik.log")

	// Allow time to startup
	time.Sleep(1 * time.Second)

	// issue SIGUSR1 signal to server process and move traefik log
	err = cmd.Process.Signal(syscall.SIGUSR1)
	c.Assert(err, checker.IsNil)
	os.Rename("traefik.log", "traefik.log.rotated")

	cmd.Process.Signal(syscall.SIGTERM)

	// Allow time for switch to be processed
	time.Sleep(3 * time.Second)

	// we have 6 lines in traefik.log.rotated
	rotated, err := os.Open("traefik.log.rotated")
	c.Assert(err, checker.IsNil)
	rotatedLog := bufio.NewScanner(rotated)
	count := 0
	for rotatedLog.Scan() {
		line := rotatedLog.Text()
		c.Log("rl " + line)
		count++
	}
	c.Assert(count, checker.Equals, 6)

	//Verify traefik.log output as expected
	file, err := os.Open("traefik.log")
	c.Assert(err, checker.IsNil)
	traefikLog := bufio.NewScanner(file)
	for traefikLog.Scan() {
		line := traefikLog.Text()
		c.Log("tl " + line)
		count++
	}
	// combined, we have 10 lines
	c.Assert(count, checker.Equals, 13)
}

func verifyEmptyErrorLog(c *check.C, name string) {
	err := try.Do(30*time.Second, func() error {
		traefikLog, e2 := ioutil.ReadFile(name)
		if e2 != nil {
			return e2
		}
		if len(traefikLog) > 0 {
			fmt.Printf("%s\n", string(traefikLog))
			c.Assert(len(traefikLog), checker.Equals, 0)
		}
		return nil
	})
	c.Assert(err, checker.IsNil)
}
