package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
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

// global variables
var (
	keys = []string{
		".ioctl", ".unlocked_ioctl", ".compat_ioctl", ".mmap", ".uring_cmd",
		".setsockopt", ".getsockopt", ".recvmsg", ".sendmsg",
	}
)

type ProgConfig struct {
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
	MaxRetry     int

	// misc configs
	Progress string
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
func createQueue(db *database.Database) ([]database.GlobalVar, error) {
	var queue []database.GlobalVar
	gvs, err := db.GetAllGlobalVar()
	if err != nil {
		return nil, fmt.Errorf("failed to create queue: %v", err)
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
	return queue, nil
}

// createBlacklist create a blacklist which includes redundant global variables, i.e. those
// whose syscall spec have existed in syzkaller
// TODO: currently we use a blacklist file to specify global variables whose spec have existed
// in syzkaller, is there a more efficient way to do this, such as querying syzkaller's database?
func createBlacklist(cfg *ProgConfig) (map[string]bool, error) {
	// read blacklist file
	blacklist := make(map[string]bool)
	data, err := os.ReadFile(cfg.BlackList)
	if err != nil {
		return nil, fmt.Errorf("failed to read blacklist file: %v\n", err)
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
func minimizeQueue(queue *[]database.GlobalVar, blacklist map[string]bool) []database.GlobalVar {
	var minimizedQueue []database.GlobalVar
	for _, gv := range *queue {
		if _, found := blacklist[gv.Name]; found {
			continue
		}
		minimizedQueue = append(minimizedQueue, gv)
	}
	return minimizedQueue
}

// setConfigs parse command-line flags and set program configurations
func setConfigs() *ProgConfig {
	// command-line flags
	var cfg ProgConfig
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
	flag.IntVar(&cfg.MaxFix, "max-fix", 5, "Maximum number of fix attempts for invalid specs")
	flag.IntVar(&cfg.MaxRetry, "max-retry", 5, "Maximum number of retry attempts for writing spec (-1 means infinite retries)")
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
			"data/prompts/fix/example_v4l2.md,",
		"Path to the fix system prompt file(s), use comma to separate multiple files",
	)
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
			scfe.err = fmt.Errorf("%s: %s: %v\n", title, path, err)
		}
	}
	fileExistHelperFunc(cfg.Env, "-env")
	fileExistHelperFunc(cfg.Db, "-db")
	fileExistHelperFunc(cfg.CheckBin, "-check-bin")
	fileExistHelperFunc(cfg.ExtractKernel, "-extract-kernel")
	vmlinuxPath := filepath.Join(cfg.CheckKernel, "vmlinux")
	fileExistHelperFunc(vmlinuxPath, "-check-kernel")
	fileExistHelperFunc(cfg.Syzkaller, "-syzkaller")
	fileExistHelperFunc(cfg.Prefix, "-prefix")
	for f := range strings.SplitSeq(cfg.OtlSysPrompt, ",") {
		fileExistHelperFunc(f, "-otl-system-prompt")
	}
	for f := range strings.SplitSeq(cfg.GenSysPrompt, ",") {
		fileExistHelperFunc(f, "-gen-system-prompt")
	}
	for f := range strings.SplitSeq(cfg.FixSysPrompt, ",") {
		fileExistHelperFunc(f, "-fix-system-prompt")
	}
	if scfe.err != nil {
		return scfe.err
	}

	// -outdir
	if cfg.Outdir == "" {
		return fmt.Errorf("-outdir: cannot be empty.")
	}

	// -syzkaller
	sc := check.SyzCheck{
		Workdir: cfg.Syzkaller,
	}
	if err := sc.CheckWorkdir(); err != nil {
		return fmt.Errorf("-syzkaller: directory is invalid: %v\n", err)
	}

	// -max-retry
	if cfg.MaxRetry < -1 {
		return fmt.Errorf("-max-retry: must be -1 or greater.")
	}

	return nil
}

// createAgent creates an agent for kernel syscal spec generation, return the agent instance and error
func createAgent(db *database.Database, sc *check.SyzCheck, cfg *ProgConfig) (*agent.Agent, error) {
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel(cfg.Model),
	)
	if err != nil {
		return nil, err
	}
	toolMap := map[string]myTools.ToolExec{
		myTools.GetFuncCodeByNameTool.Function.Name: {
			Tool: myTools.GetFuncCodeByNameTool,
			Exec: myTools.ExecGetFuncCodeByName,
		},
		myTools.GetEnumCodeByEnumeratorTool.Function.Name: {
			Tool: myTools.GetEnumCodeByEnumeratorTool,
			Exec: myTools.ExecGetEnumCodeByEnumerator,
		},
		myTools.GetEnumCodeBySpecifierTool.Function.Name: {
			Tool: myTools.GetEnumCodeBySpecifierTool,
			Exec: myTools.ExecGetEnumCodeBySpecifier,
		},
		myTools.GetStructCodeByNameTool.Function.Name: {
			Tool: myTools.GetStructCodeByNameTool,
			Exec: myTools.ExecGetStructCodeByName,
		},
		myTools.GetUnionCodeByNameTool.Function.Name: {
			Tool: myTools.GetUnionCodeByNameTool,
			Exec: myTools.ExecGetUnionCodeByName,
		},
		myTools.GetGlobalVarCodeByNameTool.Function.Name: {
			Tool: myTools.GetGlobalVarCodeByNameTool,
			Exec: myTools.ExecGetGlobalVarCodeByName,
		},
		myTools.GetTypedefCodeByDefineTool.Function.Name: {
			Tool: myTools.GetTypedefCodeByDefineTool,
			Exec: myTools.ExecGetTypedefCodeByDefine,
		},
		myTools.GetTypedefTypeByDefineTool.Function.Name: {
			Tool: myTools.GetTypedefTypeByDefineTool,
			Exec: myTools.ExecGetTypedefTypeByDefine,
		},
		myTools.GetMacroDefCodeByNameTool.Function.Name: {
			Tool: myTools.GetMacroDefCodeByNameTool,
			Exec: myTools.ExecGetMacroDefCodeByName,
		},
		myTools.GetMacroDefCodesByPatternTool.Function.Name: {
			Tool: myTools.GetMacroDefCodesByPatternTool,
			Exec: myTools.ExecGetMacroDefCodesByPattern,
		},
		myTools.GetMacroDefLocByNameTool.Function.Name: {
			Tool: myTools.GetMacroDefLocByNameTool,
			Exec: myTools.ExecGetMacroDefLocByName,
		},
	}
	toolHelper := &myTools.ToolHelper{
		Db: db,
		Sc: sc,
	}
	kAgent := agent.Agent{
		Ctx:         context.Background(),
		Model:       llm,
		Messages:    []llms.MessageContent{},
		Temperature: 0.2,
		MaxTokens:   128 << 10, // 128k
		ToolMap:     toolMap,
		ToolHelper:  toolHelper,
	}
	return &kAgent, nil
}

func main() {
	// parse flags
	cfg := setConfigs()
	stap := safeToAbsPath{err: nil}
	cfg.Env = stap.toAbsPath(cfg.Env)
	cfg.Db = stap.toAbsPath(cfg.Db)
	cfg.Outdir = stap.toAbsPath(cfg.Outdir)
	cfg.CheckBin = stap.toAbsPath(cfg.CheckBin)
	cfg.ExtractKernel = stap.toAbsPath(cfg.ExtractKernel)
	cfg.CheckKernel = stap.toAbsPath(cfg.CheckKernel)
	cfg.Syzkaller = stap.toAbsPath(cfg.Syzkaller)
	cfg.BlackList = stap.toAbsPath(cfg.BlackList)
	cfg.Prefix = stap.toAbsPath(cfg.Prefix)
	fabs := ""
	for f := range strings.SplitSeq(cfg.OtlSysPrompt, ",") {
		fabs += stap.toAbsPath(f) + ","
	}
	cfg.OtlSysPrompt = strings.TrimRight(fabs, ",")
	fabs = ""
	for f := range strings.SplitSeq(cfg.GenSysPrompt, ",") {
		fabs += stap.toAbsPath(f) + ","
	}
	cfg.GenSysPrompt = strings.TrimRight(fabs, ",")
	fabs = ""
	for f := range strings.SplitSeq(cfg.FixSysPrompt, ",") {
		fabs += stap.toAbsPath(f) + ","
	}
	cfg.FixSysPrompt = strings.TrimRight(fabs, ",")
	if stap.err != nil {
		log.Fatalf("path conversion failed: %v\n", stap.err)
	}

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

	// create a SyzCheck instances
	sc := check.SyzCheck{
		Bin:              cfg.CheckBin,
		KernelForExtract: cfg.ExtractKernel,
		KernelForCheck:   cfg.CheckKernel,
		Workdir:          cfg.Syzkaller,
	}
	if err := sc.CheckWorkdir(); err != nil {
		log.Fatalf("syzkaller workdir check failed: %v\n", err)
	}

	// create a queue that used to prompt llm for spec generation
	log.Println("Generating material queue ...")
	queue, err := createQueue(&db)
	if err != nil {
		log.Fatalf("failed to create queue: %v\n", err)
	}
	log.Printf("Original queue length: %d\n", len(queue))

	// construct blacklist and minimize the queue via blacklist to avoid redundant specification
	blacklist, err := createBlacklist(cfg)
	if err != nil {
		log.Fatalf("failed to create blacklist: %v\n", err)
	}
	queue = minimizeQueue(&queue, blacklist)
	log.Printf("Minimized queue length: %d\n", len(queue))

	// construct system prompt map, we have checked the validity of system prompt file(s) in
	// checkConfig function, so it is okay to ignore error here
	sysPromptMap := make(map[string]string)
	spmWrtFunc := func(filesWithComma string, key string) {
		files := strings.SplitSeq(filesWithComma, ",")
		var sb strings.Builder
		for file := range files {
			data, _ := os.ReadFile(file)
			sb.WriteString(string(data) + "\n")
		}
		sysPromptMap[key] = sb.String()
	}
	spmWrtFunc(cfg.OtlSysPrompt, "outline")
	spmWrtFunc(cfg.GenSysPrompt, "generate")
	spmWrtFunc(cfg.FixSysPrompt, "fix")

	// init an agent and start writing syscall specs
	kAgent, err := createAgent(&db, &sc, cfg)
	if err != nil {
		log.Fatalf("failed to create agent: %v\n", err)
	}
	buf, _ := os.ReadFile(cfg.Prefix)
	prefix := string(buf) // prefix for syz spec
	if cfg.MaxRetry == -1 {
		cfg.MaxRetry = math.MaxInt
	}
	for i, gvEntry := range queue {
		cfg.Progress = fmt.Sprintf("%d/%d", i+1, len(queue))
		fp := filepath.Join(cfg.Outdir, gvEntry.Name+"#"+cfg.Model, "spec#comp.txt")
		if _, err := os.Stat(fp); err == nil && cfg.Resume {
			log.Printf("Complete spec for global variable %s exists, reuse existing and skip generation.\n", gvEntry.Name)
			continue
		}
		spec, valid, err := writeSpec(kAgent, &sysPromptMap, &gvEntry, cfg)
		for j := 0; j < cfg.MaxRetry && err != nil; j++ {
			log.Printf("Retrying to write spec for global variable %s (attempt %d/%d) ...\n", gvEntry.Name, j+1, cfg.MaxRetry)
			spec, valid, err = writeSpec(kAgent, &sysPromptMap, &gvEntry, cfg)
		}
		if err != nil {
			log.Fatalf("failed to write spec for global variable %s: %v\n", gvEntry.Name, err)
		}
		compSpec := prefix + "\n\n" + spec
		if !valid {
			compSpec = fmt.Sprintf("# NOTE: failed to fix spec after %d attempts\n%s", cfg.MaxFix, compSpec)
		}
		err = os.WriteFile(fp, []byte(compSpec), 0644)
		if err != nil {
			log.Fatalf("failed to write final spec file: %v\n", err)
		}
		log.Printf("Spec for global variable %s has been generated successfully.\n", gvEntry.Name)
	}
}
