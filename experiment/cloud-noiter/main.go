// this variant disable iterate generation during spec generate, it give
// the reference to LLM and prompt it to get all related elements via tool calling.
// The source of these elements are directly embedded in prompt and LLM is prompted
// to generate syscall specs. The source of this file is duplicated from
// src/generator/main.go on be85413e. Please note that the implementation of this
// variant is very dirty and I only use it for quick testing. Please run this
// program under root directory of cloud :)

package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/check"
	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/pool"
	"github.com/Radon10043/cloud/src/pkg/stage"
	"github.com/Radon10043/cloud/src/pkg/utils"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// RefEntry is a struct to store reference entry for spec generation,
type RefEntry struct {
	Type string // only support "variable" and "function" currently
	Name string
}

type ProgConfig struct {
	// agent configs
	Model string
	Env   string
	Db    string

	// kernel configs
	Outdir string
	Ref    string
	Resume bool
	Prefix string

	// prompt configs
	MaxRetry int

	// misc configs
	Jobs int
}

// safeToAbsPath is a helper struct to safely convert path to absolute path
type safeToAbsPath struct {
	err error
}

// toAbsPath convert a path to absolute path
func (s *safeToAbsPath) toAbsPath(path string) string {
	if s.err != nil {
		return path
	}
	var absPath string
	absPath, s.err = filepath.Abs(path)
	return absPath
}

func main() {
	// parse flags
	cfg := setConfigs()
	if err := checkEmptyConfig(cfg); err != nil {
		log.Fatalf("invalid configuration: %v\n", err)
	}
	stap := safeToAbsPath{err: nil}
	cfg.Db = stap.toAbsPath(cfg.Db)
	cfg.Outdir = stap.toAbsPath(cfg.Outdir)
	cfg.Ref = stap.toAbsPath(cfg.Ref)

	// predefined values
	cfg.Env = stap.toAbsPath("./.env")
	cfg.Prefix = ""
	cfg.Resume = true
	cfg.MaxRetry = 5

	// check validity of flags
	if err := checkConfig(cfg); err != nil {
		log.Fatalf("invalid configuration: %v\n", err)
	}

	// load environment variables from .env file
	if err := godotenv.Load(cfg.Env); err != nil {
		log.Fatal("Error loading .env file")
	}

	// create output directory
	if err := os.MkdirAll(cfg.Outdir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v\n", err)
	}

	// connect to the database
	db := database.Database{Path: cfg.Db}
	if err := db.Connect(); err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}
	defer db.Close()

	// create a queue that used to prompt llm for spec generation
	log.Println("Generating queue ...")
	refEntries := readRefFile(cfg.Ref)
	queue, err := createQueue(&db, refEntries)
	if err != nil {
		log.Fatalf("failed to create queue: %v\n", err)
	}
	log.Printf("Queue length: %d\n", len(queue))

	if cfg.MaxRetry == -1 {
		cfg.MaxRetry = math.MaxInt
	}

	// create jobs and ress for job dispatch and result collection
	var (
		jobs  = make(chan WriteJob, len(queue))
		ress  = make(chan WriteJobRes, len(queue))
		jobWg sync.WaitGroup
		resWg sync.WaitGroup
	)
	for w := 0; w < cfg.Jobs; w++ {
		jobWg.Add(1)
		go func(tid int) {
			defer jobWg.Done()
			writeJob(tid, &db, cfg, jobs, ress)
		}(w)
	}
	resWg.Add(1)
	go func() {
		defer resWg.Done()
		for res := range ress {
			log.Printf("[T%d][%s] write job result: err=%v", res.Tid, res.Entry.GetName(), res.Err)
		}
	}()

	// start to dispatch write jobs
	for i, entry := range queue {
		jobs <- WriteJob{
			Progress: fmt.Sprintf("%d/%d", i+1, len(queue)),
			Entry:    entry,
		}
	}
	close(jobs)
	jobWg.Wait()
	close(ress)
	resWg.Wait()
	log.Printf("all write jobs have been completed.\n")
}

// checkEmptyConfig check whether essential fields in config are empty, return error if any essential field is empty
func checkEmptyConfig(cfg *ProgConfig) error {
	type helper struct {
		err error
	}
	h := helper{err: nil}
	check := func(value string, field string) {
		if h.err != nil {
			return
		}
		if value == "" {
			h.err = fmt.Errorf("%s cannot be empty", field)
		}
	}
	check(cfg.Db, "-db")
	check(cfg.Outdir, "-outdir")
	check(cfg.Ref, "-ref")
	if h.err != nil {
		return h.err
	}
	return nil
}

// readRefFile read reference entries from user-specified file, return a slice of RefEntry
func readRefFile(path string) []RefEntry {
	var entries []RefEntry
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read varlist file: %v\n", err)
	}
	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tmp := strings.Split(line, ",")
		typ, name := strings.TrimSpace(tmp[0]), strings.TrimSpace(tmp[1])
		entries = append(entries, RefEntry{Type: typ, Name: name})
	}
	return entries
}

// createQueue creates a queue of database entities from reference entries
func createQueue(db *database.Database, refEntries []RefEntry) ([]database.Entry, error) {
	var queue []database.Entry
	for _, ref := range refEntries {
		switch ref.Type {
		case "variable":
			gv, err := db.GetGlobalVar(ref.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get global variable %q from database: %v", ref.Name, err)
			}
			queue = append(queue, gv)
		case "function":
			fn, err := db.GetFunction(ref.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get function %q from database: %v", ref.Name, err)
			}
			queue = append(queue, fn)
		default:
			return nil, fmt.Errorf("unsupported reference type %q for entry %q", ref.Type, ref.Name)
		}
	}
	return queue, nil
}

// setConfigs parse command-line flags and set program configurations
func setConfigs() *ProgConfig {
	// command-line flags
	var cfg ProgConfig
	flag.StringVar(&cfg.Model, "model", "gemini-2.5-flash", "The model to use")
	flag.StringVar(&cfg.Db, "db", "", "Path to the database file")
	flag.StringVar(&cfg.Outdir, "outdir", "", "Path to the output directory")
	flag.StringVar(&cfg.Ref, "ref", "", "Path to the reference entries file")
	flag.IntVar(&cfg.Jobs, "jobs", 1, "Maximum number of parallel jobs.")
	flag.Parse()
	return &cfg
}

// checkConfig check validity of flags
func checkConfig(cfg *ProgConfig) error {
	// a helper to check file existence
	type safeCheckFileExist struct {
		err error
	}
	scfe := safeCheckFileExist{err: nil}
	fileExistHelperFunc := func(path string, title string) {
		if scfe.err != nil {
			return
		}
		_, err := os.Stat(path)
		if err != nil {
			scfe.err = fmt.Errorf("%s: %s: %v", title, path, err)
		}
	}
	fileExistHelperFunc(cfg.Env, "-env")
	fileExistHelperFunc(cfg.Db, "-db")
	if cfg.Ref != "" {
		fileExistHelperFunc(cfg.Ref, "-ref")
	}

	if scfe.err != nil {
		return scfe.err
	}

	// -jobs
	if cfg.Jobs < 1 {
		return fmt.Errorf("-jobs: must be positive")
	}

	return nil
}

type WriteJob struct {
	Progress string             // a prefix to indicate progress, e.g., "1/100"
	Entry    database.Entry     // the database entry (variable or function) to write spec for
	Db       *database.Database // database instance
}

type WriteJobRes struct {
	Tid   int            // task id
	Entry database.Entry // database entry
	Err   error          // error during writing spec
}

// writeJob start a job to write syscall spec for a database entry (variable or function)
func writeJob(tid int, db *database.Database, cfg *ProgConfig, wjs <-chan WriteJob, wjr chan<- WriteJobRes) {
	logger := log.New(os.Stdout, "[T"+strconv.Itoa(tid)+"] ", log.LstdFlags|log.Lmsgprefix)
	res := WriteJobRes{
		Tid:   tid,
		Entry: nil,
		Err:   nil,
	}

	// create an agent
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel(cfg.Model),
	)
	if err != nil {
		logger.Printf("failed to create llm instance: %v\n", err)
		res.Err = err
		wjr <- res
		return
	}

	for wj := range wjs {
		var (
			logPrefix  = fmt.Sprintf("[T%d][%s]", tid, wj.Progress)
			specPrefix = ""
			entry      = wj.Entry
		)
		res.Entry = entry

		// set prefix of spec
		if cfg.Prefix != "" {
			buf, _ := os.ReadFile(cfg.Prefix)
			specPrefix = string(buf)
		}

		// start writing spec
		specdir := filepath.Join(cfg.Outdir, "specs", entry.GetName()+"#"+cfg.Model)
		fp := filepath.Join(specdir, "spec#comp.txt")
		if _, err := os.Stat(fp); err == nil && cfg.Resume {
			logger.Printf("Complete spec for %s exists, reuse existing and skip generation.\n", entry.GetName())
			wjr <- res
			continue
		}
		spec, err := writeSpec(db, llm, nil, entry, cfg, specPrefix, logPrefix)
		for j := 0; j < cfg.MaxRetry && err != nil; j++ {
			logger.Printf(
				"Retrying to write spec for %s, err=%s (attempt %d/%d) ...\n",
				entry.GetName(), err, j+1, cfg.MaxRetry,
			)
			// sleep for a while before write spec again to avoid frequent requests
			slpTime := rand.Int31n(11) + 10
			time.Sleep(time.Duration(slpTime) * time.Second)
			spec, err = writeSpec(db, llm, nil, entry, cfg, specPrefix, logPrefix)
		}
		if err != nil {
			logger.Printf("failed to write spec for %s: %v\n", entry.GetName(), err)
			logger.Printf("Skip %s and continue with next one.\n", entry.GetName())
			res.Err = err
			wjr <- res
			continue
		}

		// write the final spec to file
		compSpec := specPrefix + "\n\n" + spec
		if err = os.WriteFile(fp, []byte(compSpec), 0644); err != nil {
			logger.Printf("failed to write final spec file: %v\n", err)
			res.Err = err
			wjr <- res
			continue
		}
		logger.Printf("Spec for %s has been generated successfully.\n", entry.GetName())

		wjr <- res
	}
}

// writeSpec start prompting agent to outline todo tasks, generate specs, and fix specs for a database entry,
// return the final syzlang spec and whether it is valid
func writeSpec(
	db *database.Database,
	llm *openai.LLM,
	sc *check.SpecCheck,
	entry database.Entry,
	cfg *ProgConfig,
	specPrefix string,
	logPrefix string,
) (string, error) {
	// init and pool default value for stage.StageHelper
	specdir := filepath.Join(cfg.Outdir, "specs", entry.GetName()+"#"+cfg.Model)
	var sh *stage.StageHelper = &stage.StageHelper{
		Workdir:    specdir,
		Next:       "outline",
		SpecPrefix: specPrefix,
		LogPrefix:  logPrefix,
		MaxFix:     0,
		Scheck:     sc,
		Tqueue:     nil,
		TqueuePath: filepath.Join(specdir, ".tqueue"),
		Spool:      &pool.SpecPool{},
		SpoolPath:  filepath.Join(specdir, ".spool"),
		Pool:       &pool.SpecPool{},
		PoolPath:   filepath.Join(specdir, ".pool"),
		Rpool:      &pool.SpecPool{},
		RpoolPath:  filepath.Join(specdir, ".rpool"),
		SyzPool:    &pool.SpecPool{},
	}
	err := os.MkdirAll(sh.Workdir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create workdir: %v", err)
	}

	// if resume is enabled, recover existing progress
	if cfg.Resume {
		if err = sh.RecoverProgress(); err != nil {
			return "", fmt.Errorf("failed to recover progress: %v", err)
		}
	} // otherwise start from scratch

	// start write spec
	spec, err := execWriteStep(db, llm, entry, sh)
	if err != nil {
		return "", fmt.Errorf("failed to exec write step: %v", err)
	}

	// Only return specs generated by agent
	return spec, nil
}

// execWriteStep execute one step of the write spec process according to sh.Next
func execWriteStep(
	db *database.Database, llm *openai.LLM, entry database.Entry, sh *stage.StageHelper,
) (string, error) {
	// retrieve all related elements first, then prompt LLM to generate specs based on these elements
	logger := log.New(os.Stdout, sh.LogPrefix+"["+entry.GetName()+"]", log.LstdFlags|log.Lmsgprefix)

	// we dont exit program if retrieve failed, instead, we check agent's message to extract related elements
	logger.Printf("Retrieving related elements ...\n")
	var relaElems strings.Builder
	kAgent := agent.NewAgent(db, llm)
	err := retrieveRelaElems(kAgent, entry, logger)
	if err != nil {
		fmt.Printf("error occured during retrieving related elements: %v\n", err)
	}
	for _, msg := range kAgent.Messages {
		if msg.Role != "tool" {
			continue
		}
		relaElems.WriteString(msg.Parts[0].(llms.ToolCallResponse).Content + "\n\n")
	}
	if err = sh.SaveQueryMessages(kAgent, "retrieve-"); err != nil {
		return "", fmt.Errorf("failed to save query messages after retrieving related elements: %v", err)
	}
	kAgent.Purge()

	// prompt LLM to generate spec based on retrieved elements
	logger.Printf("Generating spec based on retrieved related elements ...\n")
	kAgent = agent.NewAgentWithTools(nil, llm, nil)
	prompt := fmt.Sprintf("Please generate syzlang spec based on following related elements. The specification should be enclosed in code fences and the language should be syzlang.:\n```c\n%s\n\n%s\n```\n", entry.GetCode(), relaElems.String())
	kAgent.AddHumanMessage(prompt)
	resp, err := kAgent.Query()
	if err != nil {
		return "", fmt.Errorf("failed to query agent for spec generation: %v", err)
	}
	logger.Printf("Spec generation completed.\n")
	if err = sh.SaveQueryMessages(kAgent, "generate-"); err != nil {
		return "", fmt.Errorf("failed to save query messages after generating spec: %v", err)
	}
	kAgent.Purge()

	spec, found := utils.ExtractFirstCodeBlock(resp.Choices[0].Content, "syzlang")
	if !found {
		return "", fmt.Errorf("failed to extract spec from agent response")
	}

	return spec, nil
}

func retrieveRelaElems(kAgent *agent.Agent, entry database.Entry, logger *log.Logger) error {
	prompt := fmt.Sprintf(
		"please list all elements' name that related to the system calls based on following reference:"+
			"```c\n%s\n```\n",
		entry.GetCode(),
	)
	kAgent.AddHumanMessage(prompt)
	for {
		resp, err := kAgent.Query()
		if err != nil {
			return fmt.Errorf("failed to query agent: %v", err)
		}
		if len(resp.Choices[0].ToolCalls) == 0 {
			break
		}
		for _, tc := range resp.Choices[0].ToolCalls {
			logger.Printf("Tool call: %s\n", tc.FunctionCall)
		}
		err = kAgent.ExecTools()
		if err != nil {
			return fmt.Errorf("failed to execute tools: %v", err)
		}
	}
	return nil
}
