package main

import (
	"bufio"
	"flag"
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

var (
	Version   string
	BuildDate string
	Commit    string
)

type gender bool

const (
	female gender = true
	male   gender = false
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	Dark   = "\033[90m" // Dark gray
)

type uuidMap map[string]string

var thisMap = make(uuidMap, 32)

type nameConfig struct {
	gender    string
	useTitle  bool
	useFirst  bool
	useLast   bool
	formatStr string
	color     bool
}

var config nameConfig
var currentGender gender

func init() {
	flag.StringVar(&config.gender, "gender", "both", "Gender for name generation (male/female/both)")
	flag.BoolVar(&config.useTitle, "title", false, "Include title in name")
	flag.BoolVar(&config.useFirst, "first", true, "Include first name")
	flag.BoolVar(&config.useLast, "last", true, "Include last name")
	flag.StringVar(&config.formatStr, "format", "[%s]", "Format string for the name output")
	flag.BoolVar(&config.color, "color", true, "Use colors")
}

func (m uuidMap) Run(s string) (string, error) {
	for uuid, name := range m {
		if config.color {
			s = strings.ReplaceAll(s, uuid, Dark+name+Reset)
		} else {
			s = strings.ReplaceAll(s, uuid, name)
		}
	}
	return s, nil
}

func generateName() string {
	var parts []string
	if config.gender == "both" {
		currentGender = !currentGender
	}

	if config.useTitle {
		if currentGender == female {
			parts = append(parts, faker.TitleFemale())
		} else {
			parts = append(parts, faker.TitleMale())
		}
	}

	if config.useFirst {
		if currentGender == female {
			parts = append(parts, faker.FirstNameFemale())
		} else {
			parts = append(parts, faker.FirstNameMale())
		}
	}

	if config.useLast {
		parts = append(parts, faker.LastName())
	}

	return fmt.Sprintf(config.formatStr, strings.Join(parts, " "))
}

func main() {
	flag.Parse()
	switch config.gender {
	case "male":
		currentGender = male
	case "female":
		currentGender = female
	case "both":
		currentGender = female
	}

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
		if _, ok := thisMap[uuid]; !ok {
			thisMap[uuid] = generateName()
		}
	}

	return thisMap.Run(l)
}
