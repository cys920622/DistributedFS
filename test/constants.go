// This file was adapted from single_client.go provided by CPSC416 instructors.
package test

import (
	"fmt"
	"time"
	"os"
	"strings"
	"io/ioutil"
)

const (
	CHUNKNUM          = 3                           // which chunk client A will try to read from and write to
	VALID_FILE_NAME   = "cpsc416"                   // a file name client A will create
	INVALID_FILE_NAME = "invalid file;"             // a file name that the dfslib rejects
	DEADLINE          = "2018-01-29T23:59:59-08:00" // project deadline :-)
	LocalIP 		  = "127.0.0.1" // you may want to change this when testing
)

//////////////////////////////////////////////////////////////////////
// helper functions -- no need to look at these
type testLogger struct {
	prefix string
}

func NewLogger(prefix string) testLogger {
	return testLogger{prefix: prefix}
}

func (l testLogger) log(message string) {
	fmt.Printf("[%s][%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), l.prefix, message)
}

func (l testLogger) TestResult(description string, success bool) {
	var label string
	if success {
		label = "OK"
	} else {
		label = "ERROR"
	}

	l.log(fmt.Sprintf("%-70s%-10s", description, label))
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s [server-address]\n", os.Args[0])
	os.Exit(1)
}

func reportError(err error) {
	timeWarning := []string{}

	deadlineTime, _ := time.Parse(time.RFC3339, DEADLINE)
	timeLeft := deadlineTime.Sub(time.Now())
	totalHours := timeLeft.Hours()
	daysLeft := int(totalHours / 24)
	hoursLeft := int(totalHours) - 24*daysLeft

	if daysLeft > 0 {
		timeWarning = append(timeWarning, fmt.Sprintf("%d days", daysLeft))
	}

	if hoursLeft > 0 {
		timeWarning = append(timeWarning, fmt.Sprintf("%d hours", hoursLeft))
	}

	timeWarning = append(timeWarning, fmt.Sprintf("%d minutes", int(timeLeft.Minutes())-60*int(totalHours)))
	warning := strings.Join(timeWarning, ", ")

	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	fmt.Fprintf(os.Stderr, "\nPlease fix the bug above and run this test again. Time remaining before deadline: %s\n", warning)
	os.Exit(1)
}

func CleanDir() {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		fmt.Println(err)
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "client") {
			os.RemoveAll(f.Name())
		}
	}
}
//////////////////////////////////////////////////////////////////////

