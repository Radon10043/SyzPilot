package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/check"
	"github.com/Radon10043/cloud/src/pkg/database"
	osu "github.com/Radon10043/cloud/src/pkg/osutil"
	"github.com/Radon10043/cloud/src/pkg/pool"
	"github.com/Radon10043/cloud/src/pkg/stage"
	myTools "github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/joho/godotenv"
	"github.com/otiai10/copy"
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
	Os         string
	Outdir     string
	ExtractBin string
	CheckBin   string
	Kernel     string
	Ref        string
	Resume     bool
	Prefix     string // !DEPRECATED

	// spec generation configs
	Sysdir string

	// prompt configs
	OtlSysPrompt string
	GenSysPrompt string
	FixSysPrompt string
	MaxFix       int
	MaxRetry     int

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
	stap := safeToAbsPath{err: nil}
	cfg.Env = stap.toAbsPath(cfg.Env)
	cfg.Db = stap.toAbsPath(cfg.Db)
	cfg.Outdir = stap.toAbsPath(cfg.Outdir)
	cfg.ExtractBin = stap.toAbsPath(cfg.ExtractBin)
	cfg.CheckBin = stap.toAbsPath(cfg.CheckBin)
	cfg.Kernel = stap.toAbsPath(cfg.Kernel)
	cfg.Sysdir = stap.toAbsPath(cfg.Sysdir)
	if cfg.Ref != "" {
		cfg.Ref = stap.toAbsPath(cfg.Ref)
	}
	if cfg.Prefix != "" {
		cfg.Prefix = stap.toAbsPath(cfg.Prefix)
	}
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

	// check validity of sysdir
	if err := checkSysdir(cfg); err != nil {
		log.Fatalf("invalid sysdir: %v\n", err)
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

	// construct system prompt map, we have checked the validity of system prompt file(s) in
	// checkConfig function, so it is okay to ignore error here
	sysPromptMap := make(map[string]string)
	spmWrtFunc := func(filesWithComma string, key string) {
		files := strings.SplitSeq(filesWithComma, ",")
		var sb strings.Builder
		for file := range files {
			data, _ := os.ReadFile(file)
			repdata := bytes.ReplaceAll(data, []byte("{OS}"), []byte(cfg.Os))
			sb.WriteString(string(repdata))
			sb.WriteString("\n")
		}
		sysPromptMap[key] = sb.String()
	}
	spmWrtFunc(cfg.OtlSysPrompt, "outline")
	spmWrtFunc(cfg.GenSysPrompt, "generate")
	spmWrtFunc(cfg.FixSysPrompt, "fix")
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
			Progress:     fmt.Sprintf("%d/%d", i+1, len(queue)),
			Entry:        entry,
			SysPromptMap: &sysPromptMap,
		}
	}
	close(jobs)
	jobWg.Wait()
	close(ress)
	resWg.Wait()
	log.Printf("all write jobs have been completed.\n")
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
	flag.StringVar(&cfg.Model, "model", "", "The model to use")
	flag.StringVar(&cfg.Env, "env", ".env", "Path to .env file")
	flag.StringVar(&cfg.Db, "db", "", "Path to the database file")
	flag.StringVar(&cfg.Os, "os", "", "Target OS type")
	flag.StringVar(&cfg.Outdir, "outdir", "", "Path to the output directory")
	flag.StringVar(&cfg.ExtractBin, "extract-bin", "./bin/syz-extract", "Path to the syz-extract binary")
	flag.StringVar(&cfg.CheckBin, "check-bin", "./bin/syz-check", "Path to the syz-check binary")
	flag.StringVar(&cfg.Kernel, "kernel", "", "Path to kernel used for spec generation")
	flag.StringVar(&cfg.Sysdir, "sysdir", "./syzkaller/sys/", "Path to the sys directory (syzkaller/sys like structure)")
	flag.BoolVar(&cfg.Resume, "resume", true, "Whether to resume from previous interrupted run")
	flag.StringVar(&cfg.Ref, "ref", "", "Path to the reference entries file")
	flag.IntVar(&cfg.MaxFix, "max-fix", 5, "Maximum number of fix attempts for invalid specs")
	flag.IntVar(&cfg.MaxRetry, "max-retry", 5, "Maximum number of retry attempts for writing spec (-1 means infinite retries)")
	flag.IntVar(&cfg.Jobs, "jobs", 1, "Maximum number of parallel jobs.")
	flag.StringVar(
		&cfg.OtlSysPrompt,
		"otl-system-prompt",
		"./data/prompts/outline/instruction.md,"+
			"./data/prompts/outline/example_media.md,"+
			"./data/prompts/outline/example_ppp.md",
		"Path to the outline system prompt file(s), use comma to separate multiple files",
	)
	flag.StringVar(
		&cfg.GenSysPrompt,
		"gen-system-prompt",
		"./data/prompts/generate/instruction.md,"+
			"./data/prompts/generate/example_media.md,"+
			"./data/prompts/generate/example_ppp.md",
		"Path to the generate system prompt file(s), use comma to separate multiple files",
	)
	flag.StringVar(
		&cfg.FixSysPrompt,
		"fix-system-prompt",
		"./data/prompts/fix/instruction.md,"+
			"./data/prompts/fix/example_v4l2.md",
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
			scfe.err = fmt.Errorf("%s: %s: %v", title, path, err)
		}
	}
	fileExistHelperFunc(cfg.Env, "-env")
	fileExistHelperFunc(cfg.Db, "-db")
	fileExistHelperFunc(cfg.ExtractBin, "-extract-bin")
	fileExistHelperFunc(cfg.CheckBin, "-check-bin")
	fileExistHelperFunc(cfg.Sysdir, "-sysdir")
	if cfg.Prefix != "" {
		fileExistHelperFunc(cfg.Prefix, "-prefix")
	}
	if cfg.Ref != "" {
		fileExistHelperFunc(cfg.Ref, "-ref")
	}
	for f := range strings.SplitSeq(cfg.OtlSysPrompt, ",") {
		fileExistHelperFunc(f, "-otl-system-prompt")
	}
	for f := range strings.SplitSeq(cfg.GenSysPrompt, ",") {
		fileExistHelperFunc(f, "-gen-system-prompt")
	}
	for f := range strings.SplitSeq(cfg.FixSysPrompt, ",") {
		fileExistHelperFunc(f, "-fix-system-prompt")
	}

	// -os
	osType, err := osu.ParseOsType(cfg.Os)
	if err != nil {
		return fmt.Errorf("-os: %v", err)
	}

	// -kernel
	fileExistHelperFunc(cfg.Kernel, "-kernel")
	kernelObjPath := osType.KernFilePath(cfg.Kernel)
	fileExistHelperFunc(kernelObjPath, "-kernel")

	if scfe.err != nil {
		return scfe.err
	}

	// -outdir
	if cfg.Outdir == "" {
		return fmt.Errorf("-outdir: cannot be empty")
	}

	// -max-retry
	if cfg.MaxRetry < -1 {
		return fmt.Errorf("-max-retry: must be -1 or greater")
	}

	// -jobs
	if cfg.Jobs < 1 {
		return fmt.Errorf("-jobs: must be positive")
	}

	return nil
}

// checkSysdir check whether syz-extract and syz-check can be successfully executed with the cfg.Sysdir
func checkSysdir(cfg *ProgConfig) error {
	logger := log.New(os.Stdout, "[dryrun] ", log.LstdFlags|log.Lmsgprefix)
	logger.Printf("Checking validity of %v ...\n", cfg.Sysdir)
	osType, err := osu.ParseOsType(cfg.Os)
	if err != nil {
		return fmt.Errorf("failed to parse OS type: %v", err)
	}
	workdir, err := os.MkdirTemp(os.TempDir(), "check-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary workdir: %v", err)
	}
	kernelExtract, kernelCheck, err := duplicateKernel(workdir, cfg.Kernel, osType, logger)
	sc := check.NewSpecCheck(
		check.WithOs(osType),
		check.WithSyzExtract(cfg.ExtractBin),
		check.WithSyzCheck(cfg.CheckBin),
		check.WithKernelForExtract(kernelExtract),
		check.WithKernelForCheck(kernelCheck),
		check.WithWorkdir(workdir),
		check.WithSysdir(cfg.Sysdir),
	)
	sc.SetupWorkdir()
	defer os.RemoveAll(workdir)
	if stdout, stderr, valid := sc.ExtractConst(""); !valid {
		return fmt.Errorf("failed to execute syz-extract with sysdir:\n\nstdout:\n%v\n\nstderr:\n%v\n", stdout, stderr)
	}
	if stdout, stderr, valid := sc.CheckValidity(); !valid {
		return fmt.Errorf("failed to execute syz-check with sysdir:\n\nstdout:\n%v\n\nstderr:\n%v\n", stdout, stderr)
	}
	logger.Printf("%v is valid!\n", cfg.Sysdir)
	return nil
}

type WriteJob struct {
	Progress     string             // a prefix to indicate progress, e.g., "1/100"
	Entry        database.Entry     // the database entry (variable or function) to write spec for
	SysPromptMap *map[string]string // system prompt map
	Db           *database.Database // database instance
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

	osType, err := osu.ParseOsType(cfg.Os)
	if err != nil {
		logger.Printf("failed to parse OS type: %v\n", err)
		res.Err = err
		wjr <- res
		return
	}

	// create a SpecCheck instances
	wd := filepath.Join(cfg.Outdir, "instance-"+strconv.Itoa(tid))
	if _, err := os.Stat(wd); err == nil { // remove existing workdir and create a fresh one
		if err = os.RemoveAll(wd); err != nil {
			logger.Printf("failed to remove existing workdir: %v\n", err)
			res.Err = err
			wjr <- res
			return
		}
	}
	// only copy necessary dirtecoties/files for extract/check to the workdir
	if err := os.MkdirAll(wd, 0755); err != nil {
		logger.Printf("failed to create workdir: %v\n", err)
		res.Err = err
		wjr <- res
		return
	}
	kernelExtract, kernelCheck, err := duplicateKernel(wd, cfg.Kernel, osType, logger)
	if err != nil {
		logger.Printf("failed to duplicate kernel: %v\n", err)
		res.Err = err
		wjr <- res
		return
	}
	sc := check.NewSpecCheck(
		check.WithOs(osType),
		check.WithSyzExtract(cfg.ExtractBin),
		check.WithSyzCheck(cfg.CheckBin),
		check.WithKernelForExtract(kernelExtract),
		check.WithKernelForCheck(kernelCheck),
		check.WithWorkdir(wd),
		check.WithSysdir(cfg.Sysdir),
	)
	sc.SetupWorkdir()
	defer os.RemoveAll(wd)

	// create an agent
	kAgent, err := createAgent(cfg, db)
	if err != nil {
		logger.Printf("failed to create agent instance: %v\n", err)
		res.Err = err
		wjr <- res
		return
	}

	for wj := range wjs {
		var (
			logPrefix  = fmt.Sprintf("[T%d][%s]", tid, wj.Progress)
			specPrefix = ""
			entry      = wj.Entry
			spm        = wj.SysPromptMap
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
		spec, err := writeSpec(kAgent, sc, spm, entry, cfg, specPrefix, logPrefix)
		for j := 0; j < cfg.MaxRetry && err != nil; j++ {
			logger.Printf(
				"Retrying to write spec for %s, err=%s (attempt %d/%d) ...\n",
				entry.GetName(), err, j+1, cfg.MaxRetry,
			)
			// sleep for a while before write spec again to avoid frequent requests
			slpTime := rand.Int31n(11) + 10
			time.Sleep(time.Duration(slpTime) * time.Second)
			spec, err = writeSpec(kAgent, sc, spm, entry, cfg, specPrefix, logPrefix)
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

		// restore workdir for next write job, if we cannot restore workdir, subsequent jobs cannot be executed correctly,
		// so we set res.Err and return directly
		if err = sc.RestoreWorkdir(); err != nil {
			logger.Printf("failed to restore spec check workdir: %v\n", err)
			wjr <- res
			return
		}
		wjr <- res
	}
}

// createAgent creates an agent instance with the given configuration and database, return the agent instance and error
func createAgent(cfg *ProgConfig, db *database.Database) (*agent.Agent, error) {
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
	}
	kAgent := agent.NewAgent(
		agent.WithModel(llm),
		agent.WithMaxTokens(128<<10),
		agent.WithTemperature(0.2),
		agent.WithToolHelper(toolHelper),
		agent.WithToolMap(toolMap),
	)
	return kAgent, nil
}

// duplicateKernel duplicate kernel directories/files for extract/check to the workdir,
// return extract kernel path, check kernel path, and error
func duplicateKernel(prefix string, kernel string, osType osu.OsType, logger *log.Logger) (string, string, error) {
	// duplicate kernel for extract
	logger.Printf("copying extract kernel to workdir (%s) ...\n", prefix)
	src := osType.KernExtractPath(kernel)
	dst := filepath.Join(prefix, "kernel-extract")
	if err := copy.Copy(src, dst); err != nil {
		logger.Printf("failed to copy extract kernel: %v\n", err)
		return "", "", err
	}

	// run "make distclean" under kernel-extract to clean up the kernel source tree
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("make", "distclean")
	cmd.Dir = filepath.Join(prefix, "kernel-extract")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		logger.Printf("failed to run 'make distclean' under kernel-extract: %v\nstdout:\n%s\nstderr:\n%s\n", err, stdout.String(), stderr.String())
		return "", "", err
	}

	// duplicate kernel for check
	logger.Printf("copying check kernel to workdir (%s) ...\n", prefix)
	src = osType.KernFilePath(kernel)
	dst = osType.KernFilePath(filepath.Join(prefix, "kernel-check"))
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		logger.Printf("failed to create directory for check kernel: %v\n", err)
		return "", "", err
	}
	if err := copy.Copy(src, dst); err != nil {
		logger.Printf("failed to copy check kernel: %v\n", err)
		return "", "", err
	}
	return filepath.Join(prefix, "kernel-extract"), filepath.Join(prefix, "kernel-check"), nil
}

// writeSpec start prompting agent to outline todo tasks, generate specs, and fix specs for a database entry,
// return the final syzlang spec and whether it is valid
func writeSpec(
	kAgent *agent.Agent,
	sc *check.SpecCheck,
	sysPromptMap *map[string]string,
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
		MaxFix:     cfg.MaxFix,
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

	// init SyzPool with existing specs in sysdir
	specfs, err := filepath.Glob(filepath.Join(cfg.Sysdir, sc.Os.String(), "*.txt"))
	if err != nil {
		return "", fmt.Errorf("failed to glob spec files in sysdir: %v", err)
	}
	for _, specf := range specfs {
		specb, err := os.ReadFile(specf)
		if err != nil {
			return "", fmt.Errorf("failed to read spec file %s: %v", specf, err)
		}
		tmpPool, err := pool.NewSpecPoolFromSyzlang(string(specb))
		if err != nil {
			return "", fmt.Errorf("failed to convert syzlang to SpecPool for file %s: %v", specf, err)
		}
		sh.SyzPool.Merge(tmpPool)
	}

	// if resume is enabled, recover existing progress
	if cfg.Resume {
		if err = sh.RecoverProgress(); err != nil {
			return "", fmt.Errorf("failed to recover progress: %v", err)
		}
	} // otherwise start from scratch

	// start the write spec loop, limit the number of iterations to avoid infinite loop
	// TODO: for some long tasks, 100 iterations may not be enough, consider a more flexible
	// strategy to determine whether to stop the loop
	for range 100 {
		sh.UpdateNextStep()
		if sh.Next == "complete" {
			sh.Tqueue.Clear()
			sh.Spool.Clear()
			break
		}
		if err := execWriteStep(kAgent, sysPromptMap, entry, sh); err != nil {
			return "", fmt.Errorf("failed to exec write step: %v", err)
		}
		if err := sh.WriteCurrStat(); err != nil {
			return "", fmt.Errorf("failed to write current state: %v", err)
		}
	}
	if err = sh.WriteCurrStat(); err != nil {
		return "", fmt.Errorf("failed to write current state: %v", err)
	}

	if sh.Next != "complete" {
		return "", fmt.Errorf("failed to complete spec writing after max iterations")
	}

	// Only return specs generated by agent
	return sh.Pool.Syzlang(pool.WithValidComment(true)), nil
}

// execWriteStep execute one step of the write spec process according to sh.Next
func execWriteStep(
	kAgent *agent.Agent, sysPromptMap *map[string]string, entry database.Entry, sh *stage.StageHelper,
) error {
	logger := log.New(os.Stdout, sh.LogPrefix+"["+entry.GetName()+"]["+sh.Next+"] ", log.LstdFlags|log.Lmsgprefix)
	switch sh.Next {
	case "outline":
		return stage.ExecOutlineStep(kAgent, (*sysPromptMap)["outline"], entry, logger, sh)
	case "generate":
		return stage.ExecGenerateStep(kAgent, (*sysPromptMap)["generate"], entry, logger, sh)
	case "fix":
		return stage.ExecFixStep(kAgent, (*sysPromptMap)["fix"], logger, sh)
	case "complete":
	default:
		return fmt.Errorf("unknown next step: %s", sh.Next)
	}
	return nil
}
