// gjbt is a simple (temporary) wrapper for GopherJS to run tests in Chrome as
// opposed to NodeJS.
package main

import (
	"errors"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kisielk/gotool"
	"github.com/sclevine/agouti"
)

type res struct {
	Error    string
	ExitCode int
}

var (
	fTags   = flag.String("tags", "", "tags to pass to the GopherJS compiler")
	fBinary = flag.String("binary", chromeBinaryName, "path to Chrome binary")
	fDriver = flag.String("driver", "chromedriver", "path to chromedriver binary")
	fEnv    = flag.Bool("env", true, "Pass environment variables to runtime.")

	// Six flags copied almost verbatim from gopherjs.  (They start with a
	// single '-', like the go test flags.)
	fBench     = flag.String("bench", "", "Run benchmarks matching the regular expression. By default, no benchmarks run. To run all benchmarks, use '-bench=.'.")
	fBenchtime = flag.String("benchtime", "", "Run enough iterations of each benchmark to take t, specified as a time.Duration (for example, -benchtime 1h30s). The default is 1 second (1s).")
	fCount     = flag.String("count", "", "Run each test and benchmark n times (default 1). Examples are always run once.")
	fRun       = flag.String("run", "", "Run only those tests and examples matching the regular expression.")
	fShort     = flag.Bool("short", false, "Tell long-running tests to shorten their run time.")
	fVerbose   = flag.Bool("v", false, "Log all tests as they are run. Also print all text from Log and Logf calls even if the test succeeds.")

	testFailure = errors.New("test failure")
)

// TODO:
// * only works for Chrome for now
// * support verbose mode in some way

func main() {
	flag.Parse()

	if err := runChrome(); err != nil {
		handleError(err)
	}
}

func handleError(err error) {
	if err != testFailure {
		// we will have other output
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	os.Exit(1)
}

type runnerData struct {
	driver    *agouti.WebDriver
	wd        string
	tags      string
	testflags []string
}

func runChrome() error {

	pkgs := gotool.ImportPaths(flag.Args())

	wd, err := os.Getwd()
	if err != nil {
		handleError(fmt.Errorf("failed to get working directory: %v", err))
	}

	// for each package:
	//
	// 1. gopherjs test -c -o /tmp/file
	// 2. Run the below and

	opts := []agouti.Option{
		agouti.ChromeOptions(
			"args", []string{
				"headless",
				"no-default-browser-check",
				"verbose",
				"no-sandbox",
				"no-first-run",
				"disable-default-apps",
				"disable-popup-blocking",
				"disable-translate",
				"disable-background-timer-throttling",
				"disable-renderer-backgrounding",
				"disable-device-discovery-notifications",
			},
		),
		agouti.Desired(
			agouti.Capabilities{
				"loggingPrefs": map[string]string{
					"browser": "INFO",
				},
			},
		),
	}

	binaryPath := absPath(*fBinary)
	driverPath := absPath(*fDriver)

	opts = append(opts,
		agouti.ChromeOptions(
			"binary", binaryPath,
		))

	driver := agouti.NewWebDriver("http://{{.Address}}", []string{driverPath, "--port={{.Port}}"}, opts...)

	if err := driver.Start(); err != nil {
		return fmt.Errorf("failed to start driver: %v", err)
	}

	runner := &runnerData{
		driver: driver,
		wd:     wd,
		tags:   *fTags,
	}
	if *fBench != "" {
		runner.testflags = append(runner.testflags, "-test.bench", *fBench)
	}
	if *fBenchtime != "" {
		runner.testflags = append(runner.testflags, "-test.benchtime", *fBenchtime)
	}
	if *fCount != "" {
		runner.testflags = append(runner.testflags, "-test.count", *fCount)
	}
	if *fRun != "" {
		runner.testflags = append(runner.testflags, "-test.run", *fRun)
	}
	if *fShort {
		runner.testflags = append(runner.testflags, "-test.short")
	}
	if *fVerbose {
		runner.testflags = append(runner.testflags, "-test.v")
	}

	failed := false

	for _, pkg := range pkgs {
		testFail, err := runner.testPkg(pkg)
		if err != nil {
			return fmt.Errorf("error running test for %v: %v", pkg, err)
		}
		failed = failed || testFail
	}

	if err := driver.Stop(); err != nil {
		return fmt.Errorf("failed to stop driver: %v", err)
	}

	if failed {
		return testFailure
	}

	return nil
}

func (r *runnerData) testPkg(pkg string) (bool, error) {
	fmtErr := func(format string, args ...interface{}) error {
		args = append([]interface{}{pkg}, args...)
		return fmt.Errorf("pkg %v: "+format, args...)
	}

	tf, err := ioutil.TempFile("", "gjbt")
	if err != nil {
		return false, fmtErr("failed to create temp file: %v", err)
	}
	defer func() {
		n := tf.Name()
		os.Remove(n)
		os.Remove(n + ".map")
	}()

	failed := false

	bpkg, err := build.Import(pkg, r.wd, build.FindOnly)
	if err != nil {
		return false, fmtErr("failed to resolve import %v relative to %v: %v", pkg, r.wd, err)
	}

	args := []string{"test", "--tags", r.tags, "-c", "-o", tf.Name()}

	args = append(args, pkg)

	// TODO if we can/want to make these tests concurrent then
	// we will have to pass in a separate stdout and stderr
	cmd := exec.Command("gopherjs", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// this actually represents success of running the command
			// but it gave a non-zero exit code... which means the test
			// failed. stderr will have everything at this point
			return true, nil
		}

		return false, fmtErr("failed to run %v: %v", strings.Join(cmd.Args, " "), err)
	}

	test, err := ioutil.ReadFile(tf.Name())
	if err != nil {
		return false, fmtErr("failed to read from %v: %v", tf.Name(), err)
	}

	// TODO feels like we should be disposing of this resource once we're done with
	// it... especially if we end up testing lots of packages
	p, err := r.driver.NewPage()
	if err != nil {
		return false, fmtErr("failed to create new page for test: %v", err)
	}

	var ec res

	status := "ok  "
	start := time.Now()

	argsValue := append([]string{"/fake/program", "/fake/script.js"}, r.testflags...)

	var envValue = make(map[string]string)

	arguments := make(map[string]interface{})
	arguments["argv"] = argsValue
	arguments["env"] = envValue

	if *fEnv {
		for _, env := range os.Environ() {
			if index := strings.Index(env, "="); index != -1 {
				envValue[env[:index]] = env[index+1:]
			}
		}
	}

	// process.exit has to be provided, even if just a no-op, since the
	// generated gopherjs code checks for the existence of the global process
	// variable but then assumes the existence of process.exit (same for
	// process.env).
	err = p.RunScript(`
		try {
			window.process = { };
			window.process.argv = argv;
			window.process.env = env;
			window.process.exit = function(exitcode) {};
			`+string(test)+`
		}
		catch (e) {
			window.$GopherJSTestResult = {
				Error: e.stack,
				ExitCode: 1,
			};
		};
		if(typeof window.$GopherJSTestResult === 'number') {
			window.$GopherJSTestResult = {
				ExitCode: window.$GopherJSTestResult
			}
		};
		return window.$GopherJSTestResult`, arguments, &ec)

	if err != nil {
		return false, fmtErr("failed to run script: %v")
	}

	if ec.ExitCode != 0 {
		status = "FAIL"
		failed = true
	}

	logs, err := p.ReadNewLogs("browser")
	if err != nil {
		return false, fmtErr("failed to read logs: %v", err)
	}

	for _, log := range logs {
		// Format is:
		//
		// log message "console-api 4694:19 \"Success\""
		parts := strings.SplitN(log.Message, " ", 3)

		line := parts[2]

		// TODO need to understand more details on the format of the third
		// "field" - sometimes it's quoted, sometimes not
		if strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"") {
			l, err := strconv.Unquote(parts[2])
			if err != nil {
				return false, fmtErr("failed to properly parse log line output %q: %v", log.Message)
			}
			line = l
		}

		// We output to stdout for now
		fmt.Println(line)
	}

	if ec.Error != "" {
		fmt.Fprintln(os.Stderr, ec.Error)
	}
	fmt.Printf("%s\t%s\t%.3fs\n", status, bpkg.ImportPath, time.Since(start).Seconds())

	return failed, nil
}

func absPath(s string) string {
	if strings.Index(s, string(filepath.Separator)) != -1 {
		b, err := filepath.Abs(s)
		if err != nil {
			handleError(fmt.Errorf("failed to make %v absolute", s))
		}
		return b
	}

	b, err := exec.LookPath(s)
	if err != nil {
		handleError(fmt.Errorf("Failed to find \"%v\" in your PATH environment variable.", s))
	}
	return b
}
