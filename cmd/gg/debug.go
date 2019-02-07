package main

import (
	"fmt"
	"os"
	"time"
)

// TODO tidy up the distinction between tracing and timing. Perhaps they belong
// as a single concept

func debugf(format string, args ...interface{}) {
	if format[len(format)-1] != '\n' {
		format += "\n"
	}
	if *fDebug {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

func logTiming(format string, args ...interface{}) {
	ms := func(pre, post time.Time) int64 {
		return int64(post.Sub(pre) / time.Millisecond)
	}
	if *fTraceTime {
		now := time.Now()
		fmt.Fprintf(tabber, "%v\t %v\t - %v\n", ms(startTime, now), ms(lastTime, now), fmt.Sprintf(format, args...))
		lastTime = now
	}
}

func logTrace(format string, args ...interface{}) {
	if format[len(format)-1] != '\n' {
		format += "\n"
	}
	if *fTrace {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
