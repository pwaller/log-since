package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Read exactly one line
func ReadLine(r io.ReadSeeker) (l string, err error) {
	pos, err := r.Seek(0, os.SEEK_CUR)
	check(err)

	lr := bufio.NewScanner(r)
	lr.Scan()

	_, err = r.Seek(pos+int64(len(lr.Bytes())), os.SEEK_SET)
	check(err)

	return lr.Text(), lr.Err()
}

// Finds the whoele line at the current position of `r` by seeking backwards.
// Afterwards, r is positioned at the start of the line
func FindlineAt(r io.ReadSeeker) (int64, string) {
	// Size of window to search backwards in
	var windowSize int64 = 256

	data := make([]byte, windowSize)

	currentPos, err := r.Seek(0, os.SEEK_CUR)
	check(err)

	for {
		if currentPos < windowSize {
			// We're as far left as we can go
			windowSize = currentPos
		}
		_, err := r.Seek(currentPos-windowSize, os.SEEK_SET)
		check(err)
		currentPos -= windowSize

		n, err := r.Read(data[:windowSize])
		check(err)
		if int64(n) < windowSize {
			log.Panicf("n < windowSize : %v < %v", n, windowSize)
		}

		pos := bytes.LastIndex(data, []byte("\n")) + 1
		if pos > 0 {
			final, err := r.Seek(-(windowSize - int64(pos)), os.SEEK_CUR)
			check(err)

			content, err := ReadLine(r)
			check(err)

			return final, content
		}
	}
}

func SearchFile(r io.ReadSeeker, startTime time.Time) {

	fileLength, err := r.Seek(0, os.SEEK_END)
	check(err)

	// Mapping of file coordinate onto position of closest
	// newline on the left of that point
	visitedIndices := map[int64]int64{}
	// Mapping whether each newline is before or after
	pointAfter := map[int64]bool{}

	sort.Search(int(fileLength), func(n int) bool {
		_, err := r.Seek(int64(n), os.SEEK_SET)
		check(err)
		pos, line := FindlineAt(r)
		t := ParseNginxTime(line)
		result := t.After(startTime)
		pointAfter[pos] = result
		for i, _ := range line {
			visitedIndices[pos+int64(i)] = pos
		}
		return result
	})

	_, err = r.Seek(1, os.SEEK_CUR)
	check(err)
}

var bracket = func(r rune) bool { return strings.ContainsRune("[]", r) }

func ParseNginxTime(l string) time.Time {
	const timeSpec = "02/Jan/2006:15:04:05 -0700"

	fields := strings.FieldsFunc(l, bracket)
	t, err := time.Parse(timeSpec, fields[1])
	check(err)
	return t
}

func main() {
	fd, err := os.Open(os.Args[1])
	check(err)
	defer fd.Close()

	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	SearchFile(fd, twoHoursAgo)

	lineReader := bufio.NewScanner(fd)
	for lineReader.Scan() {
		l := lineReader.Text()
		fmt.Fprintln(os.Stdout, l)
	}
}
