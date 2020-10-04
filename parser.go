package main

import (
	"bufio"
	"database/sql"
	"github.com/cheggaaa/pb/v3"
	"log"
	"os"
	"strings"
	"sync"
)

func startParse(db *sql.DB) {
	parseRoute(db)
	parseStopTime(db)
}

func parseRoute(db *sql.DB) {
	file, err := os.Open("./data/routes.txt")
	count, _ := lineCount("./data/routes.txt")

	bar := pb.Simple.Start(count)

	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		phrase := scanner.Text()
		phraseSplit := strings.Split(phrase, ",")
		insertRoute(db, phraseSplit[0], phraseSplit[2])
		bar.Increment()
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	bar.Finish()
}

func parseStopTime(db *sql.DB) {
	file, err := os.Open("./data/stop_times.txt")
	count, _ := lineCount("./data/stop_times.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	bar := pb.Simple.Start(count)
	scanner := bufio.NewScanner(file)

	type entry struct {
		tripID        string
		routeID       string
		arrivalTime   string
		departureTime string
		stopID        string
		stopHeadsign  string
		wg            *sync.WaitGroup
	}
	entries := make(chan entry)
	wg := sync.WaitGroup{}

	go func() {
		for {
			select {
			case entry, ok := <-entries:
				if ok {
					insertStopTime(db, entry.tripID, entry.routeID, entry.arrivalTime, entry.departureTime, entry.stopID, entry.stopHeadsign)
					bar.Increment()
					entry.wg.Done()
				}
			}
		}
	}()

	linesChunkLen := 64 * 1024
	lines := make([]string, 0, 0)

	for scanner.Scan() {
		phrase := scanner.Text()
		lines = append(lines, phrase)

		if len(lines) == linesChunkLen {
			wg.Add(len(lines))
			process := lines
			go func() {
				for _, text := range process {
					lineSplit := strings.Split(text, ",")
					tripID := lineSplit[0]
					parseTripID := strings.Split(tripID, ".")
					routeID := ""
					if len(parseTripID) > 2 {
						routeID = parseTripID[2]
					}

					e := entry{wg: &wg}
					e.tripID = tripID
					e.routeID = routeID
					e.arrivalTime = lineSplit[1]
					e.departureTime = lineSplit[2]
					e.stopID = lineSplit[3]
					e.stopHeadsign = lineSplit[5]
					entries <- e
				}
			}()
			lines = make([]string, 0, linesChunkLen)
		}
	}
	wg.Wait()
	close(entries)
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	bar.Finish()
}
