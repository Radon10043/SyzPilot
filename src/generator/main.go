package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/joho/godotenv"
)

var (
	// command-line flags
	flagModel = flag.String("model", "gpt-5-nano-ca", "The model to use")
	flagEnv   = flag.String("env", ".env", "Path to .env file")
	flagDb    = flag.String("db", "", "Path to the database file")

	keys = []string{".ioctl", ".unlocked_ioctl"} // global variable keys of interest
)

// check if any key is found in the code
func keyFound(code string) bool {
	for _, key := range keys {
		if strings.Contains(code, key) {
			return true
		}
	}
	return false
}

// generate material queue from database
func genMaterialQueue(dbPath string) ([]string, error) {
	var queue []string
	db := database.Database{Path: dbPath}
	err := db.Connect()
	if err != nil {
		return queue, err
	}
	gvs, err := db.GetAllGlobalVar()
	if err != nil {
		return queue, err
	}
	for _, gv := range gvs {
		if gv.Code == "" {
			continue
		}
		if !keyFound(gv.Code) {
			continue
		}
		queue = append(queue, gv.Name)
	}
	db.Close()
	return queue, nil
}

func main() {
	flag.Parse()

	// load environment variables from .env file
	err := godotenv.Load(*flagEnv)
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// create a queue that used to prompt llm for spec generation
	fmt.Println("Generating material queue ...")
	_, err = genMaterialQueue(*flagDb)
	if err != nil {
		log.Fatalf("Failed to generate material queue: %v", err)
	}

	// TODO: filter data in queue to avoid redundancy, e.g. skip ioctls whose spec
	// have existed in syzkaller. Is there a efficiency way to filter them?

	// TODO: prompting agent to generate spec, agent can use tools to query database,
	// verify validity of spec, etc. For now, we hope to have following tools:
	// - query function/struct/union/enum/typedef code by their names
	// - call syz-check of syzkaller to verify spec validity

	// TODO: output generated spec
}
