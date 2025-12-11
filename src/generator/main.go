package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Radon10043/cloud/src/generator/agent"
	"github.com/Radon10043/cloud/src/generator/check"
	"github.com/Radon10043/cloud/src/generator/database"
	myTools "github.com/Radon10043/cloud/src/generator/tools"
	"github.com/Radon10043/cloud/src/generator/utils"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type progConfig struct {
	// agent configs
	Model string
	Env   string
	Db    string

	// kernel configs
	Outdir        string
	CheckBin      string
	ExtractKernel string
	CheckKernel   string
	Syzkaller     string
	BlackList     string
	Resume        bool
	Prefix        string

	// prompt configs
	OtlSysPrompt string
	GenSysPrompt string
	FixSysPrompt string
	MaxFix       int
}

// global variables
var (
	keys = []string{
		".ioctl", ".unlocked_ioctl", ".compat_ioctl", ".mmap", ".uring_cmd",
		".setsockopt", ".getsockopt", ".recvmsg", ".sendmsg",
	}
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

// createBlacklist create a blacklist which includes redundant global variables, i.e. those
// whose syscall spec have existed in syzkaller
// TODO: currently we use a blacklist file to specify global variables whose spec have existed
// in syzkaller, is there a more efficient way to do this, such as querying syzkaller's database?
func createBlacklist(cfg *progConfig) (map[string]bool, error) {
	// read blacklist file
	blacklist := make(map[string]bool)
	data, err := os.ReadFile(cfg.BlackList)
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
	return blacklist, nil
}

// minimizeQueue minimize the queue by removing global variables in blacklist
func minimizeQueue(queue *[]database.GlobalVar, blacklist map[string]bool) ([]database.GlobalVar, error) {
	var minimizedQueue []database.GlobalVar
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

// writeSpec start prompting agent to outline todo tasks, generate specs, and fix specs for a global variable
// return the final syzlang spec and whether it is valid
func writeSpec(
	kAgent *agent.Agent, sysPromptMap *map[string]string, gvEntry *database.GlobalVar, sc *check.SyzCheck, cfg *progConfig,
) (string, bool) {
	logger := log.New(os.Stdout, "["+gvEntry.Name+"][outline] ", log.LstdFlags|log.Lmsgprefix)
	outline := collectOutline(kAgent, (*sysPromptMap)["outline"], gvEntry, cfg, logger)
	logger = log.New(os.Stdout, "["+gvEntry.Name+"][generate] ", log.LstdFlags|log.Lmsgprefix)
	jsonSpec := collectSpec(kAgent, (*sysPromptMap)["generate"], gvEntry, cfg, outline, logger)
	logger = log.New(os.Stdout, "["+gvEntry.Name+"][fix] ", log.LstdFlags|log.Lmsgprefix)
	syzSpec, valid := fixSpec(kAgent, (*sysPromptMap)["fix"], sc, gvEntry, cfg, jsonSpec, logger)
	return syzSpec, valid
}

// fixSpec start a loop to fix invalid syscall spec, also with the help of agent
func fixSpec(
	kAgent *agent.Agent, sysPrompt string, sc *check.SyzCheck, gvEntry *database.GlobalVar, cfg *progConfig, jsonSpec string, logger *log.Logger,
) (string, bool) {
	var (
		spec  string
		found bool
	)
	fdir := filepath.Join(cfg.Outdir, gvEntry.Name+"#"+cfg.Model)
	_ = os.MkdirAll(fdir, 0755)
	fp := filepath.Join(fdir, "spec.txt")

	// if spec.txt exists and resume is enabled, reuse existing spec
	if _, err := os.Stat(fp); err == nil && cfg.Resume {
		logger.Printf("Spec file exists, reuse existing.\n")
		buf, err := os.ReadFile(fp)
		if err != nil {
			logger.Fatalf("failed to read spec file: %v\n", err)
		}
		spec = string(buf)
	} else { // otherwise, convert json spec to syzlang and write to spec.txt
		spec, err = utils.Json2syzlang(jsonSpec)
		if err != nil {
			logger.Fatalf("failed to convert json spec to syzlang: %v\n", err)
		}
		err = os.WriteFile(fp, []byte(spec), 0644)
		if err != nil {
			logger.Fatalf("failed to write spec file: %v\n", err)
		}
	}

	// make agent ready for fix loop
	kAgent.CleanMessages()
	err := kAgent.AddSystemMessage(sysPrompt)
	if err != nil {
		logger.Fatalf("failed to add system prompt to agent: %v\n", err)
	}

	// check validity of spec and prompt agent to fix it if invalid
	buf, err := os.ReadFile(cfg.Prefix)
	if err != nil {
		logger.Fatalf("failed to read prefix file: %v\n", err)
	}
	prefix := string(buf)
	valid := false
	for i := 0; i < cfg.MaxFix; i++ { // limit the number of fix attempts
		logger.Printf("Checking validity of spec ...\n")
		if spec == "" {
			logger.Fatalf("empty spec, stop.\n")
		}
		stdout, stderr, err := checkSpecValidity(sc, prefix+"\n\n"+spec)
		if err == nil {
			logger.Printf("Spec is valid!\n")
			valid = true
			break
		}
		logger.Printf(
			"Spec is invalid, trying to fix.\n"+
				"========== stdout ==========\n%s\n"+
				"========== stderr ==========\n%s\n",
			stdout.String(), stderr.String(),
		)
		specBlock := fmt.Sprintf("```syzlang\n%s\n```\n", spec)
		errmsgBlock := fmt.Sprintf("```\n%s\n%s\n```\n", stdout.String(), stderr.String())
		kAgent.AddHumanMessage(specBlock + "\n" + errmsgBlock)
		response := fixSpecLoop(kAgent, logger)
		spec, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "syzlang")
		if !found {
			logger.Fatalf("failed to extract syzlang code fence from fix response.\n")
		}
		err = os.WriteFile(fp, []byte(spec), 0644)
		if err != nil {
			logger.Fatalf("failed to write spec file: %v\n", err)
		}
	}

	return spec, valid
}

// fixSpecLoop run a loop to fix invalid syscall spec
func fixSpecLoop(kAgent *agent.Agent, logger *log.Logger) *llms.ContentResponse {
	var (
		response *llms.ContentResponse
		err      error
	)

	// prompt agent to fix spec
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

	return response
}

// collectSpec prompt agent to generate syscall spec or reuse existing spec for a global variable
func collectSpec(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, cfg *progConfig, outline string, logger *log.Logger,
) string {
	fdir := filepath.Join(cfg.Outdir, gvEntry.Name+"#"+cfg.Model)
	_ = os.MkdirAll(fdir, 0755)
	fp := filepath.Join(fdir, "spec.json")

	// if spec.json exist and resume is enabled, reuse existing spec
	if _, err := os.Stat(fp); err == nil && cfg.Resume {
		logger.Printf("Spec file exists, reuse existing.\n")
		buf, err := os.ReadFile(fp)
		if err != nil {
			logger.Fatalf("failed to read spec file: %v\n", err)
		}
		outline = string(buf)
	}
	outlineMap := make(map[string]any)
	err := json.Unmarshal([]byte(outline), &outlineMap)
	if err != nil {
		logger.Fatalf("failed to parse outline json: %v\n", err)
	}

	// prompt agent to generate spec iteratively until all todo tasks are done
	var found bool
	todoNum := len(outlineMap["todo"].([]any))
	for todoNum > 0 {
		kAgent.CleanMessages()
		response := genSpec(kAgent, sysPrompt, gvEntry, outline, logger)
		outline, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "json")
		if !found {
			logger.Fatalf("failed to extract json code fence from generate response.\n")
		}
		_, err := utils.Json2syzlang(outline)
		if err != nil {
			logger.Fatalf("failed to generate valid json: %v\n", err)
		}
		err = os.WriteFile(fp, []byte(outline), 0644)
		if err != nil {
			logger.Fatalf("failed to write spec file: %v\n", err)
		}
		err = json.Unmarshal([]byte(outline), &outlineMap)
		if err != nil {
			logger.Fatalf("failed to parse outline json: %v\n", err)
		}
		todoNum = len(outlineMap["todo"].([]any))
	}
	logger.Printf("All todo tasks are done.\n")

	return outline
}

// genSpec prompt agent to generate syscall spec iteratively
func genSpec(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, outline string, logger *log.Logger,
) *llms.ContentResponse {
	var (
		response *llms.ContentResponse
		err      error
	)

	// make agent ready for spec generation stage
	err = kAgent.AddSystemMessage(sysPrompt)
	if err != nil {
		logger.Fatalf("failed to add system prompt to agent: %v\n", err)
	}

	// prompt agent to generate syscall spec
	humanMsg := fmt.Sprintf("```c\n%s\n```\n\n```json\n%s```\n", gvEntry.Code, outline)
	kAgent.AddHumanMessage(humanMsg)
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
	logger.Printf("Generation loop stop reason: %v", response.Choices[0].StopReason)

	return response
}

// collectOutline prompt agent to outline todo tasks or reuse existing outline for a global variable
func collectOutline(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, cfg *progConfig, logger *log.Logger,
) string {
	var (
		outline string
		found   bool
	)
	fdir := filepath.Join(cfg.Outdir, gvEntry.Name+"#"+cfg.Model)
	_ = os.MkdirAll(fdir, 0755)
	fp := filepath.Join(fdir, "outline.json")

	// if outline.json doesn't exist or resume is disabled, prompt agent to outline todo tasks
	if _, err := os.Stat(fp); err != nil || !cfg.Resume {
		kAgent.CleanMessages()
		response := genOutline(kAgent, sysPrompt, gvEntry, logger)
		outline, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "json")
		if !found {
			logger.Fatalf("failed to extract json code fence from outline response.\n")
		}
		_, err := utils.Json2syzlang(outline)
		if err != nil {
			logger.Fatalf("failed to generate valid json: %v\n", err)
		}
		err = os.WriteFile(fp, []byte(outline), 0644)
		if err != nil {
			logger.Fatalf("failed to write outline file: %v\n", err)
		}
	} else {
		logger.Printf("Outline file exists, reuse existing and skip outline stage.\n")
		data, err := os.ReadFile(fp)
		if err != nil {
			logger.Fatalf("failed to read outline file: %v\n", err)
		}
		outline = string(data)
	}

	return outline
}

// genOutline prompt agent to outline todo tasks for a global variable
func genOutline(kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, logger *log.Logger) *llms.ContentResponse {
	var (
		response *llms.ContentResponse
		err      error
	)

	// make agent ready for outline stage
	err = kAgent.AddSystemMessage(sysPrompt)
	if err != nil {
		logger.Fatalf("failed to add system prompt to agent: %v\n", err)
	}

	// prompt agent to outline todo tasks
	kAgent.AddHumanMessage(gvEntry.Code)
	for {
		response, err = kAgent.Query()
		if err != nil {
			logger.Fatalf("failed to run agent in outline stage: %v\n", err)
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
			logger.Fatalf("failed to execute tools in outline stage: %v\n", err)
		}
	}

	return response
}

// setConfigs parse command-line flags and set program configurations
func setConfigs() *progConfig {
	// command-line flags
	var cfg progConfig
	flag.StringVar(&cfg.Model, "model", "gemini-2.5-pro", "The model to use")
	flag.StringVar(&cfg.Env, "env", ".env", "Path to .env file")
	flag.StringVar(&cfg.Db, "db", "", "Path to the database file")
	flag.StringVar(&cfg.Outdir, "outdir", "", "Path to the output directory")
	flag.StringVar(&cfg.CheckBin, "check-bin", "bin/syz-check", "Path to the syz-check binary")
	flag.StringVar(&cfg.ExtractKernel, "extract-kernel", "", "Path to kernel used for spec extraction")
	flag.StringVar(&cfg.CheckKernel, "check-kernel", "", "Path to kernel used for spec checking")
	flag.StringVar(&cfg.Syzkaller, "syzkaller", "syzkaller/", "Path to the syzkaller directory")
	flag.StringVar(&cfg.BlackList, "blacklist", "data/blacklist.txt", "Path to the global variable blacklist file")
	flag.BoolVar(&cfg.Resume, "resume", true, "Whether to resume from previous interrupted run")
	flag.StringVar(&cfg.Prefix, "prefix", "data/prefix.txt", "Path to the prefix file for syscall syz spec")
	flag.IntVar(&cfg.MaxFix, "max-fix", 10, "Maximum number of fix attempts for invalid specs")
	flag.StringVar(
		&cfg.OtlSysPrompt,
		"otl-system-prompt",
		"data/prompts/outline/instruction.md,"+
			"data/prompts/outline/example_media.md,"+
			"data/prompts/outline/example_ppp.md",
		"Path to the outline system prompt file(s), use comma to separate multiple files",
	)
	flag.StringVar(
		&cfg.GenSysPrompt,
		"gen-system-prompt",
		"data/prompts/generate/instruction.md,"+
			"data/prompts/generate/example_media.md,"+
			"data/prompts/generate/example_ppp.md",
		"Path to the generate system prompt file(s), use comma to separate multiple files",
	)
	flag.StringVar(
		&cfg.FixSysPrompt,
		"fix-system-prompt",
		"data/prompts/fix/instruction.md,"+
			"data/prompts/fix/example_media.md,"+
			"data/prompts/fix/example_ppp.md",
		"Path to the fix system prompt file(s), use comma to separate multiple files",
	)
	flag.Parse()
	return &cfg
}

// checkConfig check validity of flags
func checkConfig(cfg *progConfig) {
	// check validity of -env
	_, err := os.Stat(cfg.Env)
	if err != nil {
		log.Fatalf("failed to get stat of env file: %v\n", err)
	}

	// -db
	_, err = os.Stat(cfg.Db)
	if err != nil {
		log.Fatalf("failed to get stat of database file: %v\n", err)
	}

	// -outdir
	if cfg.Outdir == "" {
		log.Fatalf("output directory cannot be empty.")
	}

	// -check-bin
	_, err = os.Stat(cfg.CheckBin)
	if err != nil {
		log.Fatalf("failed to get stat of syz-check binary: %v\n", err)
	}

	// -extract-kernel
	_, err = os.Stat(cfg.ExtractKernel)
	if err != nil {
		log.Fatalf("failed to get stat of extract kernel source: %v\n", err)
	}

	// -check-kernel
	vmlinuxPath := filepath.Join(cfg.CheckKernel, "vmlinux")
	_, err = os.Stat(vmlinuxPath)
	if err != nil {
		log.Fatalf("failed to get stat of vmlinux in check kernel: %v\n", err)
	}

	// -syzkaller
	sc := check.SyzCheck{
		Workdir: cfg.Syzkaller,
	}
	if err := sc.CheckWorkdir(); err != nil {
		log.Fatalf("syzkaller's directory is invalid: %v\n", err)
	}

	// -prefix
	_, err = os.Stat(cfg.Prefix)
	if err != nil {
		log.Fatalf("failed to get stat of prefix file: %v\n", err)
	}

	// -otl-system-prompt
	otlSysPromptFiles := strings.SplitSeq(cfg.OtlSysPrompt, ",")
	for f := range otlSysPromptFiles {
		_, err = os.Stat(f)
		if err != nil {
			log.Fatalf("failed to get stat of system prompt file %v\n", err)
		}
	}

	// -gen-system-prompt
	genSysPromptFiles := strings.SplitSeq(cfg.GenSysPrompt, ",")
	for f := range genSysPromptFiles {
		_, err = os.Stat(f)
		if err != nil {
			log.Fatalf("failed to get stat of system prompt file %v\n", err)
		}
	}

	// -fix-system-prompt
	fixSysPromptFiles := strings.SplitSeq(cfg.FixSysPrompt, ",")
	for f := range fixSysPromptFiles {
		_, err = os.Stat(f)
		if err != nil {
			log.Fatalf("failed to get stat of system prompt file %v\n", err)
		}
	}
}

func main() {
	// parse flags and check their validity
	cfg := setConfigs()
	cfg.Env = toAbsPath(cfg.Env)
	cfg.Db = toAbsPath(cfg.Db)
	cfg.Outdir = toAbsPath(cfg.Outdir)
	cfg.CheckBin = toAbsPath(cfg.CheckBin)
	cfg.ExtractKernel = toAbsPath(cfg.ExtractKernel)
	cfg.CheckKernel = toAbsPath(cfg.CheckKernel)
	cfg.Syzkaller = toAbsPath(cfg.Syzkaller)
	checkConfig(cfg)

	// load environment variables from .env file
	var err error
	err = godotenv.Load(cfg.Env)
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// create output directory
	err = os.MkdirAll(cfg.Outdir, 0755)
	if err != nil {
		log.Fatalf("failed to create output directory: %v\n", err)
	}

	// connect to the database
	db := database.Database{Path: cfg.Db}
	err = db.Connect()
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}
	defer db.Close()
	myTools.DB = &db

	// create a SyzCheck instances
	sc := check.SyzCheck{
		Bin:              cfg.CheckBin,
		KernelForExtract: cfg.ExtractKernel,
		KernelForCheck:   cfg.CheckKernel,
		Workdir:          cfg.Syzkaller,
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

	// construct blacklist and minimize the queue via blacklist to avoid redundant specification
	blacklist, err := createBlacklist(cfg)
	if err != nil {
		log.Fatalf("failed to create blacklist: %v\n", err)
	}
	queue, err = minimizeQueue(&queue, blacklist)
	if err != nil {
		log.Fatalf("failed to minimize material queue: %v\n", err)
	}
	log.Printf("Minimized material queue length: %d\n", len(queue))

	// construct system prompt map, we have checked the validity of system prompt file(s) in
	// checkConfig function, so it is okay to ignore error here
	sysPromptMap := make(map[string]string)
	spmWrtFunc := func(filesWithComma string, key string) {
		files := strings.SplitSeq(filesWithComma, ",")
		var sb strings.Builder
		for file := range files {
			data, _ := os.ReadFile(toAbsPath(file))
			sb.WriteString(string(data) + "\n")
		}
		sysPromptMap[key] = sb.String()
	}
	spmWrtFunc(cfg.OtlSysPrompt, "outline")
	spmWrtFunc(cfg.GenSysPrompt, "generate")
	spmWrtFunc(cfg.FixSysPrompt, "fix")

	// init agent and start writing syscall specs
	buf, _ := os.ReadFile(cfg.Prefix)
	prefix := string(buf) // prefix for syz spec
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel(cfg.Model),
	)
	if err != nil {
		log.Fatalf("failed to create llm: %v\n", err)
	}
	kAgent := agent.Agent{
		Ctx:         context.Background(),
		Model:       llm,
		Tools:       myTools.ToolList,
		Messages:    []llms.MessageContent{},
		Temperature: 0.2,
	}
	for _, gvEntry := range queue {
		fp := filepath.Join(cfg.Outdir, gvEntry.Name+"#"+cfg.Model, "spec#comp.txt")
		if _, err := os.Stat(fp); err == nil && cfg.Resume {
			log.Printf("Complete spec for global variable %s exists, reuse existing and skip generation.\n", gvEntry.Name)
			continue
		}
		spec, valid := writeSpec(&kAgent, &sysPromptMap, &gvEntry, &sc, cfg)
		compSpec := prefix + "\n\n" + spec
		if !valid {
			compSpec = fmt.Sprintf("# NOTE: failed to fix spec after %d attempts\n%s", cfg.MaxFix, compSpec)
		}
		err := os.WriteFile(fp, []byte(compSpec), 0644)
		if err != nil {
			log.Fatalf("failed to write final spec file: %v\n", err)
		}
		log.Printf("Spec for global variable %s has been generated successfully.\n", gvEntry.Name)
	}
}
