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

	"github.com/cilium/statedb"
	"github.com/cilium/statedb/index"
	"github.com/go-faker/faker/v4"
)

var (
	db            *statedb.DB
	uuidTable     statedb.RWTable[*UUIDNamePair]
	providedFlags parameterFlags
	currentGender gender

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

type UUIDNamePair struct {
	ID   string
	Name string
}

// Define how to index and query the object.
var UUIDIndex = statedb.Index[*UUIDNamePair, string]{
	Name: "id",
	FromObject: func(obj *UUIDNamePair) index.KeySet {
		return index.NewKeySet(index.String(obj.ID))
	},
	FromKey: func(id string) index.Key {
		return index.String(id)
	},
	Unique: true,
}

var NameIndex = statedb.Index[*UUIDNamePair, string]{
	Name: "name",
	FromObject: func(obj *UUIDNamePair) index.KeySet {
		return index.NewKeySet(index.String(obj.Name))
	},
	FromKey: func(name string) index.Key {
		return index.String(name)
	},
	Unique: true,
}

type parameterFlags struct {
	gender      string
	useTitle    bool
	useFirst    bool
	useLast     bool
	formatStr   string
	color       bool
	showVersion bool
}

func init() {
	flag.StringVar(&providedFlags.gender, "gender", "both", "Gender for name generation (male/female/both)")
	flag.BoolVar(&providedFlags.useTitle, "title", false, "Include title in name")
	flag.BoolVar(&providedFlags.useFirst, "first", true, "Include first name")
	flag.BoolVar(&providedFlags.useLast, "last", true, "Include last name")
	flag.StringVar(&providedFlags.formatStr, "format", "[%s]", "Format string for the name output")
	flag.BoolVar(&providedFlags.color, "color", true, "Use colors")
	flag.BoolVar(&providedFlags.showVersion, "version", false, "Print version information")

	createDatabase()
}

func printVersion() {
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("BuildDate: %s\n", BuildDate)
	fmt.Printf("Commit: %s\n", Commit)
}

func generateName() string {
	var parts []string
	if providedFlags.gender == "both" {
		currentGender = !currentGender
	}

	if providedFlags.useTitle {
		if currentGender == female {
			parts = append(parts, faker.TitleFemale())
		} else {
			parts = append(parts, faker.TitleMale())
		}
	}

	if providedFlags.useFirst {
		if currentGender == female {
			parts = append(parts, faker.FirstNameFemale())
		} else {
			parts = append(parts, faker.FirstNameMale())
		}
	}

	if providedFlags.useLast {
		parts = append(parts, faker.LastName())
	}

	return fmt.Sprintf(providedFlags.formatStr, strings.Join(parts, " "))
}

func main() {
	flag.Parse()
	switch providedFlags.gender {
	case "male":
		currentGender = male
	case "female":
		currentGender = female
	case "both":
		currentGender = female
	}

	if providedFlags.showVersion {
		printVersion()
		os.Exit(0)
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
			fmt.Fprintf(os.Stderr, "Ktnxbye!\n")
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
		if _, ok := fetchByUUID(uuid); !ok {
			var name string
			var exitCounter int
			for {
				name = generateName()
				_, ok := fetchByName(name)
				if ok {
					exitCounter++
					if exitCounter > 1000 {
						log.Fatal("Could not generate a random name in 1000 loops. Out of names...")
					}
					continue
				}
				break
			}
			storeInDb(name, uuid)
		}
	}

	return Run(l)
}

func fetchByUUID(uuid string) (obj *UUIDNamePair, ok bool) {
	txn := db.ReadTxn()
	obj, _, ok = uuidTable.Get(txn, UUIDIndex.Query(uuid))
	return obj, ok
}

func fetchByName(name string) (obj *UUIDNamePair, ok bool) {
	txn := db.ReadTxn()
	obj, _, ok = uuidTable.Get(txn, NameIndex.Query(name))
	return obj, ok
}

func storeInDb(name, uuid string) {
	// commit in a transaction to stateDB
	wtxn := db.WriteTxn(uuidTable)
	if _, _, err := uuidTable.Insert(wtxn, &UUIDNamePair{uuid, name}); err != nil {
		log.Fatal(err)
	}
	wtxn.Commit()
}

func Run(s string) (string, error) {
	// Iterate over all objects
	txn := db.ReadTxn()
	for obj := range uuidTable.All(txn) {
		if providedFlags.color {
			s = strings.ReplaceAll(s, obj.ID, Dark+obj.Name+Reset)
		} else {
			s = strings.ReplaceAll(s, obj.ID, obj.Name)
		}
	}
	return s, nil
}

// Create the database and the table.
func createDatabase() {
	var err error
	db = statedb.New()
	uuidTable, err = statedb.NewTable(
		"uuid_and_users",
		UUIDIndex,
		NameIndex,
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.RegisterTable(uuidTable); err != nil {
		log.Fatal(err)
	}
}
