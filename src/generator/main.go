package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Radon10043/cloud/src/generator/agent"
	"github.com/Radon10043/cloud/src/generator/check"
	"github.com/Radon10043/cloud/src/generator/database"
	myTools "github.com/Radon10043/cloud/src/generator/tools"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var (
	// command-line flags
	flagModel         = flag.String("model", "gemini-2.5-pro", "The model to use")
	flagEnv           = flag.String("env", ".env", "Path to .env file")
	flagDb            = flag.String("db", "", "Path to the database file")
	flagSystemPrompt  = flag.String("system-prompt", "data/prompts/system.md", "Path to the system prompt file")
	flagOutdir        = flag.String("outdir", "", "Path to the output directory")
	flagCheckBin      = flag.String("check-bin", "bin/syz-check", "Path to the syz-check binary")
	flagExtractKernel = flag.String("extract-kernel", "", "Kernel version used for spec extraction")
	flagCheckKernel   = flag.String("check-kernel", "", "Kernel version used for spec checking")
	flagSyzkaller     = flag.String("syzkaller", "syzkaller/", "Path to the syzkaller directory")
	flagBlackList     = flag.String("blacklist", "data/blacklist.txt", "Path to the global variable blacklist file")

	// global variable keys of interest
	keys = []string{
		".ioctl", ".unlocked_ioctl", ".compat_ioctl", ".mmap", ".uring_cmd",
		".setsockopt", ".getsockopt", ".recvmsg", ".sendmsg",
	}
	// TODO: if ioctl functions are analyzed, skip it to avoid redundancy
)

// toAbsaPath convert a path to absolute path
func toAbsPath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("abs path conversion failed: %v", err)
	}
	return absPath
}

// keyFound check if any key is found in the code
func keyFound(code string) bool {
	for _, key := range keys {
		if strings.Contains(code, key) {
			return true
		}
	}
	return false
}

// checkFlags check validity of flags
func checkFlags() {
	_, err := os.Stat(*flagEnv)
	if err != nil {
		log.Fatalf("failed to get stat of env file: %v\n", err)
	}
	_, err = os.Stat(*flagDb)
	if err != nil {
		log.Fatalf("failed to get stat of database file: %v\n", err)
	}
	_, err = os.Stat(*flagSystemPrompt)
	if err != nil {
		log.Fatalf("failed to get stat of system prompt file %v\n", err)
	}
	if *flagOutdir == "" {
		log.Fatalf("output directory cannot be empty.")
	}
	_, err = os.Stat(*flagCheckBin)
	if err != nil {
		log.Fatalf("failed to get stat of syz-check binary: %v\n", err)
	}
	_, err = os.Stat(*flagExtractKernel)
	if err != nil {
		log.Fatalf("failed to get stat of extract kernel source: %v\n", err)
	}
	vmlinuxPath := filepath.Join(*flagCheckKernel, "vmlinux")
	_, err = os.Stat(vmlinuxPath)
	if err != nil {
		log.Fatalf("failed to get stat of vmlinux in check kernel: %v\n", err)
	}
	sc := check.SyzCheck{
		Workdir: *flagSyzkaller,
	}
	if err := sc.CheckWorkdir(); err != nil {
		log.Fatalf("syzkaller's directory is invalid: %v\n", err)
	}
}

// createQueue creates a queue includes global variables that includes interested keys
func createQueue(db *database.Database) []database.GlobalVar {
	var queue []database.GlobalVar
	gvs, err := db.GetAllGlobalVar()
	if err != nil {
		log.Fatalf("failed to get all global variables: %v", err)
	}
	for _, gv := range gvs {
		if gv.Code == "" {
			continue
		}
		if !keyFound(gv.Code) {
			continue
		}
		queue = append(queue, gv)
	}
	return queue
}

// minimizeQueue minimize the queue by removing redundant global variables, i.e. those
// whose syscall spec have existed in syzkaller.
// TODO: currently we use a blacklist file to filter redundant global variables, is there
// a more efficient way to do this?
func minimizeQueue(queue *[]database.GlobalVar) ([]database.GlobalVar, error) {
	var minimizedQueue []database.GlobalVar
	blacklist := make(map[string]bool)
	data, err := os.ReadFile(*flagBlackList)
	if err != nil {
		log.Fatalf("failed to read blacklist file: %v\n", err)
	}
	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		blacklist[line] = true
	}
	for _, gv := range *queue {
		if _, found := blacklist[gv.Name]; found {
			continue
		}
		minimizedQueue = append(minimizedQueue, gv)
	}
	return minimizedQueue, nil
}

// checkSpecValidity check the validity of a syscall spec, return whether it is valid and error message if any
func checkSpecValidity(sc *check.SyzCheck, spec string) (bytes.Buffer, bytes.Buffer, error) {
	err := sc.CleanWorkdir()
	if err != nil {
		log.Fatalf("failed to clean syzkaller workdir: %v\n", err)
	}
	err = sc.AddSpec(spec)
	if err != nil {
		log.Fatalf("failed to add spec to syzkaller workdir: %v\n", err)
	}
	stdout, stderr, err := sc.ExtractConst()
	if err != nil {
		return stdout, stderr, err
	}
	stdout, stderr, err = sc.CheckValidity()
	if err != nil {
		return stdout, stderr, err
	}
	return stdout, stderr, nil
}

// genSpecLoop start a loop to generate syscall spec based on a global variable
func genSpecLoop(kAgent *agent.Agent, gvEntry *database.GlobalVar, sc *check.SyzCheck) *llms.ContentResponse {
	logger := log.New(os.Stdout, "["+gvEntry.Name+"] ", log.LstdFlags|log.Lmsgprefix)
	logger.Printf("Starting to generate specs \n")
	kAgent.CleanMessages()
	err := kAgent.AddSystemMessage(kAgent.SystemPrompt)
	if err != nil {
		logger.Fatalf("failed to add system prompt to agent: %v\n", err)
	}
	kAgent.AddHumanMessage(gvEntry.Code)
	var response *llms.ContentResponse = nil
	// generation loop
	for {
		logger.Printf("Query agent ...\n")
		response, err = kAgent.Query()
		if err != nil {
			logger.Fatalf("failed to run agent: %v\n", err)
		}
		logger.Printf("AI Response: %s\n", response.Choices[0].Content)
		if len(response.Choices[0].ToolCalls) == 0 {
			break
		}
		for _, tc := range response.Choices[0].ToolCalls {
			logger.Printf("Tool Call: %v\n", tc.FunctionCall)
		}
		err = kAgent.ExecTools()
		if err != nil {
			logger.Fatalf("failed to execute tools: %v\n", err)
		}
	}
	log.Printf("Generation loop stop reason: %v", response.Choices[0].StopReason)
	// fix loop
	for {
		logger.Printf("Checking validity of spec ...\n")
		spec := response.Choices[0].Content
		if spec == "" {
			logger.Fatalf("empty spec, stop.\n")
		}
		spec = strings.Replace(spec, "```syzlang", "", 1)
		spec = strings.Replace(spec, "```", "", 1)
		stdout, stderr, err := checkSpecValidity(sc, spec)
		if err == nil {
			logger.Printf("Spec is valid!\n")
			break
		}
		logger.Printf("Spec is invalid, trying to fix ...\n")
		kAgent.AddHumanMessage(stdout.String() + "\n\n" + stderr.String())
		for {
			response, err = kAgent.Query()
			if err != nil {
				logger.Fatalf("failed to run agent in fix loop: %v\n", err)
			}
			logger.Printf("AI Response: %s\n", response.Choices[0].Content)
			if len(response.Choices[0].ToolCalls) == 0 {
				break
			}
			for _, tc := range response.Choices[0].ToolCalls {
				logger.Printf("Tool Call: %v\n", tc.FunctionCall)
			}
			err = kAgent.ExecTools()
			if err != nil {
				logger.Fatalf("failed to execute tools in fix loop: %v\n", err)
			}
		}
	}
	return response
}

func main() {
	// parse flags and check their validity
	flag.Parse()
	checkFlags()
	*flagEnv = toAbsPath(*flagEnv)
	*flagDb = toAbsPath(*flagDb)
	*flagOutdir = toAbsPath(*flagOutdir)
	*flagCheckBin = toAbsPath(*flagCheckBin)
	*flagExtractKernel = toAbsPath(*flagExtractKernel)
	*flagCheckKernel = toAbsPath(*flagCheckKernel)
	*flagSyzkaller = toAbsPath(*flagSyzkaller)

	// load environment variables from .env file
	var err error
	err = godotenv.Load(*flagEnv)
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// create output directory
	err = os.MkdirAll(*flagOutdir, 0755)
	if err != nil {
		log.Fatalf("failed to create output directory: %v\n", err)
	}

	// connect to the database
	db := database.Database{Path: *flagDb}
	err = db.Connect()
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}
	defer db.Close()
	myTools.DB = &db

	// create a SyzCheck instances
	sc := check.SyzCheck{
		Bin:              *flagCheckBin,
		KernelForExtract: *flagExtractKernel,
		KernelForCheck:   *flagCheckKernel,
		Workdir:          *flagSyzkaller,
	}
	err = sc.CheckWorkdir()
	if err != nil {
		log.Fatalf("syzkaller workdir check failed: %v\n", err)
	}
	myTools.SC = &sc

	// create a queue that used to prompt llm for spec generation
	log.Println("Generating material queue ...")
	queue := createQueue(&db)
	log.Printf("Material queue length: %d\n", len(queue))

	// minimize the queue to avoid redundant specification
	queue, err = minimizeQueue(&queue)
	if err != nil {
		log.Fatalf("failed to minimize material queue: %v\n", err)
	}
	log.Printf("Minimized material queue length: %d\n", len(queue))

	// construct prompt template for agent, we have checked the validity of system prompt file in
	// checkFlags function, so it is okay to ignore error here
	sysPrompt, _ := os.ReadFile(*flagSystemPrompt)
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel(*flagModel),
	)
	if err != nil {
		log.Fatalf("failed to create llm: %v\n", err)
	}
	kAgent := agent.Agent{
		SystemPrompt: string(sysPrompt),
		Ctx:          context.Background(),
		Model:        llm,
		Tools:        myTools.ToolList,
		Messages:     []llms.MessageContent{},
		Temperature:  0.2,
	}
	for _, gvEntry := range queue[:1] { // TODO: remove [:1] to process all entries
		gvEntry, _ = db.GetGlobalVar("_ctl_fops")
		response := genSpecLoop(&kAgent, &gvEntry, &sc)
		log.Printf("Final response: %s\n", response.Choices[0].Content)
		spec := response.Choices[0].Content
		spec = strings.Replace(spec, "```syzlang", "", 1)
		spec = strings.Replace(spec, "```", "", 1)
		var (
			fn string // file name
			fp string // file path
		)
		// save spec and messages
		fn = gvEntry.Name + "#" + *flagModel + ".txt"
		fp = filepath.Join(*flagOutdir, fn)
		os.WriteFile(fp, []byte(spec), 0644)
		fn = gvEntry.Name + "#" + *flagModel + ".msg"
		fp = filepath.Join(*flagOutdir, fn)
		kAgent.SaveMessages(fp)
	}
}
