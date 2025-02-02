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

var db *statedb.DB
var uuidTable statedb.RWTable[*uuidMapV2]

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

type uuidMapV2 struct {
	ID   string
	Name string
}

// Define how to index and query the object.
var UUIDIndex = statedb.Index[*uuidMapV2, string]{
	Name: "id",
	FromObject: func(obj *uuidMapV2) index.KeySet {
		return index.NewKeySet(index.String(obj.ID))
	},
	FromKey: func(id string) index.Key {
		return index.String(id)
	},
	Unique: true,
}

var thisMap = make(uuidMap, 32)

type nameConfig struct {
	gender      string
	useTitle    bool
	useFirst    bool
	useLast     bool
	formatStr   string
	color       bool
	showVersion bool
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
	flag.BoolVar(&config.showVersion, "version", false, "Print version information")

	createDatabase()
}

func printVersion() {
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("BuildDate: %s\n", BuildDate)
	fmt.Printf("Commit: %s\n", Commit)
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

	if config.showVersion {
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
			name := generateName()
			storeInDb(name, uuid)
		}
	}

	// TODO justin je bent hier! hij doet wel store & search maar je runt nog iets
	// leegs van die oude map[string]string.
	return Run(l)
	// return thisMap.Run(l)
}

func fetchByUUID(uuid string) (obj *uuidMapV2, ok bool) {
	txn := db.ReadTxn()
	obj, _, ok = uuidTable.Get(txn, UUIDIndex.Query(uuid))
	return obj, ok
}

func storeInDb(name, uuid string) {
	// commit in a transaction to stateDB
	wtxn := db.WriteTxn(uuidTable)
	uuidTable.Insert(wtxn, &uuidMapV2{uuid, name})
	wtxn.Commit()

}

func Run(s string) (string, error) {
	// Iterate over all objects
	txn := db.ReadTxn()
	for obj := range uuidTable.All(txn) {
		if config.color {
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
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.RegisterTable(uuidTable); err != nil {
		log.Fatal(err)
	}

	/* in the functhingie
	wtxn := db.WriteTxn(uuidTable)

	// Insert some objects
	myObjects.Insert(wtxn, &MyObject{1, "a"})
	myObjects.Insert(wtxn, &MyObject{2, "b"})
	myObjects.Insert(wtxn, &MyObject{3, "c"})


	if feelingLucky {
	  // Commit the changes.
	  wtxn.Commit()
	}
	*/

	/*
	  // Query the objects with a snapshot of the database.
	  txn := db.ReadTxn()

	  if obj, _, found := myObjects.Get(txn, IDIndex.Query(1)); found {
	    ...
	  }

	  // Iterate over all objects
	  for obj := range myObjects.All() {
	    ...
	  }

	  // Iterate with revision
	  for obj, revision := range myObjects.All() {
	    ...
	  }

	  // Iterate all objects and then wait until something changes.
	  objs, watch := myObjects.AllWatch(txn)
	  for obj := range objs { ... }
	  <-watch

	  // Grab a new snapshot to read the new changes.
	  txn = db.ReadTxn()

	  // Iterate objects with ID >= 2
	  objs, watch = myObjects.LowerBoundWatch(txn, IDIndex.Query(2))
	  for obj := range objs { ... }

	  // Iterate objects where ID is between 0x1000_0000 and 0x1fff_ffff
	  objs, watch = myObjects.PrefixWatch(txn, IDIndex.Query(0x1000_0000))
	  for obj := range objs { ... }
	*/
}
