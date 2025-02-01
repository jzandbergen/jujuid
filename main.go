package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/go-faker/faker/v4"
)

// id:name
type uuidMap map[string]string

var thisMap = make(uuidMap, 32)

func (m uuidMap) Run(s string) (string, error) {
	for uuid, name := range m {
		s = strings.ReplaceAll(s, uuid, name)
	}
	return s, nil
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM)
	go func() {
		s := <-sigs
		fmt.Fprintf(os.Stderr, "Caught signal: %v", s)
		os.Exit(1)
	}()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		ok := scanner.Scan()
		if !ok {
			fmt.Printf("Ktnxbye!\n")
			os.Exit(0)
		}
		err := scanner.Err()
		if err == io.EOF {
			os.Exit(0)
		}
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}

		bytes := scanner.Bytes()
		line := string(bytes)
		output, err := processLine(line)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", output)
	}
}

func processLine(l string) (result string, err error) {
	// b56d2ce4-484d-49bb-89cb-da4517df6c66
	pattern, err := regexp.Compile("[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	uuids := pattern.FindAllString(l, -1)
	for _, uuid := range uuids {
		var name string
		name, ok := thisMap[uuid]
		if !ok {
			// TODO function
			name = fmt.Sprintf("[UUID: %s %s %s]",
				faker.TitleMale(),
				faker.FirstNameMale(),
				faker.LastName())
			thisMap[uuid] = name
		}
	}

	return thisMap.Run(l)
}
