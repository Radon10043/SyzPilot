package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/check"
	"github.com/Radon10043/cloud/src/pkg/fidl"
	osu "github.com/Radon10043/cloud/src/pkg/osutil"
	"github.com/Radon10043/cloud/src/pkg/pool"
	myTools "github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/Radon10043/cloud/src/pkg/utils"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var (
	fidlsyzHacker = NewHacker(
		WithHackPatterns(map[string]func(string) string{
			`array\[\]`:                     func(string) string { return "array[int8, 114514]" },
			`(?m)^([\t ]+)resource([\t ]+)`: func(string) string { return "${1}reso${2}" },
			`(?m)^([\t ]+)parent([\t ]+)`:   func(string) string { return "${1}pare${2}" },
			`(?m)(\w+) \{[\n\t ]+\}`:        func(string) string { return "$1 {\n\tyjsp\tvoid\n}" },
			`(?m)^(?:[\t ]+)fidl_union_member\[(\d+),(?:\s+)\]`: func(string) string {
				return "    yjsp_" + uuid.New().String()[:8] + " fidl_union_member[$1, array[int8, 1919810]]"
			},
		}),
		WithUnhackPatterns(map[string]func(string) string{
			`array\[int8, 114514\]`:          func(string) string { return "array[]" },
			`(?m)(\w+) \{\n\tyjsp\tvoid\n\}`: func(string) string { return "$1 {\n\t}" },
			`(?m)^([ \t]*)yjsp_[a-zA-Z0-9_-]+[ \t]+fidl_union_member\[(\d+),[ \t]*array\[int8,[ \t]*1919810\]\]`: func(string) string {
				return "${1}fidl_union_member[$2, ]"
			},
		}),
	)
)

func main() {
	// parse program configs
	cfg := setConfigs()
	if err := cfg.checkEmpty(); err != nil {
		log.Fatalf("check empty failed: %v\n", err)
	}
	if err := cfg.toAbs(); err != nil {
		log.Fatalf("convert to absolute paths failed: %v\n", err)
	}
	if err := cfg.checkValid(); err != nil {
		log.Fatalf("check validity of configs failed: %v\n", err)
	}

	// after compilation, the fuchsia repository is too large (232G on my machine),
	// duplicate the original repository is very expensive. So, if user sets -kernel
	// to the possible original fuchsia repository, prompt to use scripts/fuchsia/refine.sh
	// to create a minimized one
	size, err := dirsize(cfg.Kernel)
	if err != nil {
		log.Fatalf("failed to calculate directory size of %v: %v\n", cfg.Kernel, err)
	}
	if size >= (1 << 34) {
		log.Fatalf("%v is too large (>= 16GB), consider to use scripts/fuchsia/refine.sh to create a minimized one.", cfg.Kernel)
	}

	// other preparations
	if err := cfg.checkSysdir(); err != nil {
		log.Fatalf("check validity of sysdir failed: %v\n", err)
	}
	if err := godotenv.Load(cfg.Env); err != nil {
		log.Fatalf("failed to load .env: %v\n", err)
	}
	if err := os.MkdirAll(cfg.Outdir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v\n", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.Outdir, "logs"), 0755); err != nil {
		log.Fatalf("failed to create log directory: %v\n", err)
	}

	// create the SpecCheck instance
	logger := log.New(os.Stdout, "[setup] ", log.LstdFlags|log.Lmsgprefix)
	workdir := filepath.Join(cfg.Outdir, "instance")
	if _, err := os.Stat(workdir); err == nil {
		os.RemoveAll(workdir)
	}
	if err := os.MkdirAll(workdir, 0755); err != nil {
		log.Fatalf("failed to create %v: %v\n", workdir, err)
	}
	defer os.RemoveAll(workdir)
	kernelExtract, kernelCheck, err := duplicateKernel(workdir, cfg.Kernel, osu.Fuchsia, logger)
	if err != nil {
		log.Fatalf("failed to duplicate kernel: %v\n", err)
	}
	sc := check.NewSpecCheck(
		check.WithOs(osu.Fuchsia),
		check.WithSyzExtract(cfg.ExtractBin),
		check.WithSyzCheck(cfg.CheckBin),
		check.WithKernelForExtract(kernelExtract),
		check.WithKernelForCheck(kernelCheck),
		check.WithWorkdir(workdir),
		check.WithSysdir(cfg.Sysdir),
	)
	sc.SetupWorkdir()
	logger.Printf("workdir setup completed.")

	// create the agent
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel(cfg.Model),
	)
	if err != nil {
		log.Fatalf("failed to init llm: %v\n", err)
	}

	// format runtime arch
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}

	// load fidl IR (*.fidl.json) and expose it to the agent through the fidl query tools,
	// so the agent can consult the original FIDL definitions while fixing specs. The IR
	// files ship inside the fuchsia source tree under out/(x64|arm64)/fidling/gen/sdk/fidl.
	fidljsonPattern := filepath.Join(cfg.Kernel, "out", arch, "fidling", "gen", "sdk", "fidl", "*", "*.fidl.json")
	logger.Printf("loading fidl IR from %v ...", fidljsonPattern)
	fidlIndex, err := fidl.Load(fidljsonPattern)
	if err != nil {
		log.Fatalf("failed to load fidl IR: %v\n", err)
	}
	logger.Printf("loaded %v fidl declarations.", fidlIndex.Len())
	kAgent := agent.NewAgent(
		agent.WithModel(llm),
		agent.WithToolMap(myTools.FidlToolMap()),
		agent.WithToolHelper(&myTools.ToolHelper{Fidl: fidlIndex}),
	)

	// start to fix specs
	err = fixFidlsyzSpec(cfg, kAgent, sc)
	if err != nil {
		log.Fatalf("failed to fix fidl syz spec: %v\n", err)
	}
}

// fixFidlsyzSpec attempts to continue fixing specs under fixed directory or fix them from scratch,
// returns error if failed to fix.
func fixFidlsyzSpec(cfg *ProgConfig, kAgent *agent.Agent, sc *check.SpecCheck) error {
	// do some preparation for fixing
	poos, err := prepareFix(cfg)
	if err != nil {
		return fmt.Errorf("failed to prepare fixing: %v", err)
	}

	// fix error nodes in poos in loop and agentic manner
	if err := fixLoop(cfg, poos, sc, kAgent); err != nil {
		return fmt.Errorf("run fix loop failed: %v", err)
	}

	log.Printf("fix completed, using bin/pool2syz to convert pools under %v back to syzlang specs", filepath.Join(cfg.Outdir, "fixed"))
	return nil
}

// fixLoop fixes specs in poos in an agentic and loop manner, it returns error message if failed to fix
func fixLoop(cfg *ProgConfig, poos map[string]*pool.SpecPool, sc *check.SpecCheck, kAgent *agent.Agent) error {
	sysPrompt, err := loadSysPrompt(cfg.FixSysPrompt)
	if err != nil {
		return fmt.Errorf("failed to load fix system prompt: %v", err)
	}

	// Iteratively collect error nodes and prompt the agent to fix them, re-checking
	// after each round. The loop keeps running across rounds until every error node
	// is either fixed or has exhausted attempt budget. Within a round the nodes are
	// fixed concurrently by cfg.Jobs workers (see fixErrorNodes).
	for round := 1; round <= cfg.MaxFix; round++ {
		logger := log.New(os.Stdout, "[fix] ", log.LstdFlags|log.Lmsgprefix)
		errNodes, err := catchErrorNodes(poos, sc)
		if err != nil {
			return fmt.Errorf("failed to catch error nodes: %v", err)
		}
		if len(errNodes) == 0 {
			logger.Printf("no error nodes found")
			break
		}
		logger.Printf("round %v/%v: %d error node(s) to be fixed", round, cfg.MaxFix, len(errNodes))
		fixed := fixErrorNodes(errNodes, kAgent, sysPrompt, cfg)

		// deduplicate elems after each fix activity
		if err := sc.RestoreWorkdir(); err != nil {
			return fmt.Errorf("failed to restore %v: %v", sc.Workdir, err)
		}

		// persist progress so a resumed run picks up the fixes already applied
		if err := savePools(poos); err != nil {
			return fmt.Errorf("failed to save pools to the local: %v", err)
		}
		logger.Printf("applied %v/%v fix(es)", fixed, len(errNodes))
	}
	return nil
}

// fixErrorNodes fixes nodes who has error in agentic manner, using up to jobs
// concurrent workers. Each worker owns a cloned agent so the workers do not race
// on the shared agent's message history. Every ErrorNode points to a distinct
// SpecElement, so writing fixes back in parallel is safe. Returns the number of
// nodes that successfully fixed.
func fixErrorNodes(errNodes []ErrorNode, kAgent *agent.Agent, sysPrompt string, cfg *ProgConfig) int {
	jobs := min(cfg.Jobs, len(errNodes))

	var (
		fixed   atomic.Int64
		wg      sync.WaitGroup
		indexCh = make(chan int)
	)
	worker := func(kAgent *agent.Agent, tid int) {
		defer wg.Done()
		for i := range indexCh {
			node := errNodes[i]
			logger := log.New(
				os.Stdout,
				fmt.Sprintf("[T%v][%v/%v][%v] ", tid, i+1, len(errNodes), node.Elem.Name),
				log.LstdFlags|log.Lmsgprefix,
			)
			if err := fixErrorNode(kAgent, node, sysPrompt, logger, cfg); err != nil {
				logger.Printf("fix node \"%v\" failed: err=%v, skip it", node.Elem.Name, err)
				continue
			}
			fixed.Add(1)
		}
	}

	for job := range jobs {
		wg.Add(1)
		go worker(kAgent.Clone(), job+1)
	}
	for i := range errNodes {
		indexCh <- i
	}
	close(indexCh)
	wg.Wait()

	return int(fixed.Load())
}

// fixErrorNode prompts the agent to fix a single error node, with the fidl query tools
// available, and updates the node's code in place with the corrected syzlang.
func fixErrorNode(kAgent *agent.Agent, node ErrorNode, sysPrompt string, logger *log.Logger, cfg *ProgConfig) error {
	kAgent.Purge()
	if err := kAgent.AddSystemMessage(sysPrompt); err != nil {
		return fmt.Errorf("failed to add system prompt: %v", err)
	}

	var b strings.Builder
	b.WriteString("The following syzlang node is invalid:\n\n")
	b.WriteString("```syzlang\n")
	b.WriteString(strings.TrimRight(node.Elem.Code, "\n"))
	b.WriteString("\n```\n\n")
	b.WriteString("Reported errors (line:column: message):\n```\n")
	b.WriteString(strings.Join(node.Issues, "\n"))
	b.WriteString("\n```\n\n")
	b.WriteString("Use the fidl query tools to consult the original FIDL definition behind this node, " +
		"then reply with the corrected node in a single ```syzlang code block. Keep the node name unchanged.")
	kAgent.AddHumanMessage(b.String())

	logger.Printf("node has %v issues ...", len(node.Issues))
	var response *llms.ContentResponse
	var err error
	for {
		response, err = kAgent.Query()
		if err != nil {
			return fmt.Errorf("failed to query agent: %v", err)
		}
		logger.Printf("AI Response: %q\n", response.Choices[0].Content)
		if len(response.Choices[0].ToolCalls) == 0 {
			break
		}
		for _, tc := range response.Choices[0].ToolCalls {
			logger.Printf("Tool Call: %v\n", tc.FunctionCall)
		}
		err = kAgent.ExecTools()
		if err != nil {
			return fmt.Errorf("failed to execute tools: %v", err)
		}
	}

	// save query messages
	msgPath := filepath.Join(cfg.Outdir, "fixed", fmt.Sprintf("fix-%v.msg", time.Now().UnixMilli()))
	if err := kAgent.SaveMessageFile(msgPath); err != nil {
		return fmt.Errorf("failed to save query messages: %v", err)
	}

	fixed, found := utils.ExtractFirstCodeBlock(response.Choices[0].Content, "syzlang")
	if !found {
		return fmt.Errorf("no syzlang code block found in agent response")
	}
	fixed = strings.TrimSpace(fixed)
	if fixed == "" {
		return fmt.Errorf("agent returned an empty fix")
	}

	node.Elem.Code = fixed
	node.Elem.Valid = false
	return nil
}

// prepareFix does some preparations for fixing, returns a map (key is json file path and
// value is corresponding SpecPool) and error messages
func prepareFix(cfg *ProgConfig) (map[string]*pool.SpecPool, error) {
	logger := log.New(os.Stdout, "[prepare] ", log.LstdFlags|log.Lmsgprefix)

	// create fixed directory
	fixedDir := filepath.Join(cfg.Outdir, "fixed")
	if _, err := os.Stat(fixedDir); err != nil || !cfg.Resume {
		logger.Printf("setup fixed directory ...")
		os.RemoveAll(fixedDir)
		if err := os.MkdirAll(fixedDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create fixed directory: %v", err)
		}
		if err := createPrelPools(cfg.Sysdir, cfg.FidlsyzDir, fixedDir); err != nil {
			return nil, fmt.Errorf("failed to create preliminary pools for fidl syz sepcs: %v", err)
		}
	}

	// read json files under fixed directory to recover progress or start from scratch
	logger.Printf("recover progress from fixed directory ...")
	poos, err := recoverPools(fixedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to recover pools: %v", err)
	}

	return poos, nil
}

// savePools writes every pool back to its json file (the map key is the file path).
func savePools(poos map[string]*pool.SpecPool) error {
	for path, po := range poos {
		jstr, err := po.Json()
		if err != nil {
			return fmt.Errorf("failed to convert pool to json: %v", err)
		}
		if err := os.WriteFile(path, []byte(jstr), 0644); err != nil {
			return fmt.Errorf("failed to write pool to file: %v", err)
		}
	}
	return nil
}

type ErrorNode struct {
	Elem   *pool.SpecElement
	Issues []string
}

func catchErrorNodes(poos map[string]*pool.SpecPool, sc *check.SpecCheck) ([]ErrorNode, error) {
	// restore the workdir on return so this function can be called repeatedly
	// across fix iterations without colliding with previously written spec files.
	defer sc.RestoreWorkdir()

	// add all specs to sysdir, remembering which spec file maps to which pool so
	// we can resolve error locations back to the owning pool later.
	poolByFile := make(map[string]*pool.SpecPool)
	for k, v := range poos {
		filename := strings.TrimSuffix(filepath.Base(k), ".json")
		if _, err := sc.AddSpecWithName(v.Syzlang(), filename); err != nil {
			return nil, fmt.Errorf("failed to add %v to sysdir: %v", filename, err)
		}
		poolByFile[filename] = v
	}

	stdout, stderr, valid := sc.ExtractConst("")
	if !valid {
		locs := append(catchErrorLocations(stdout), catchErrorLocations(stderr)...)
		return mapErrorsToNodes(poolByFile, locs), nil
	}

	stdout, stderr, valid = sc.CheckValidity()
	if !valid {
		locs := append(catchErrorLocations(stdout), catchErrorLocations(stderr)...)
		return mapErrorsToNodes(poolByFile, locs), nil
	}

	return nil, nil
}

// elemSpan is the 1-based line range [start, end] occupied by a SpecElement
// inside the written spec file.
type elemSpan struct {
	start int
	end   int
	elem  *pool.SpecElement
}

// mapErrorsToNodes resolves each error location to the SpecElement that owns the
// reported line, merging errors that point to the same element into one ErrorNode.
//
// The mapping is done purely by string matching against the spec text instead of
// parsing it, so it still works when the generated spec contains syntax errors
// (which is exactly when we need to locate the faulty node).
func mapErrorsToNodes(poos map[string]*pool.SpecPool, locs []ErrorLocation) []ErrorNode {
	// cache each pool's element line ranges, computed lazily per spec file
	spansCache := make(map[string][]elemSpan)
	spansOf := func(file string, po *pool.SpecPool) []elemSpan {
		if spans, ok := spansCache[file]; ok {
			return spans
		}
		// Reconstruct the exact text written to the spec file; line numbers here
		// match the ones reported by syz-extract.
		text := po.Syzlang()
		var spans []elemSpan
		for _, elem := range *po {
			start, end := lineRangeOf(text, elem.Code)
			if start == 0 { // code block not found, skip
				continue
			}
			spans = append(spans, elemSpan{start: start, end: end, elem: elem})
		}
		spansCache[file] = spans
		return spans
	}

	var order []*pool.SpecElement
	byElem := make(map[*pool.SpecElement]*ErrorNode)
	for _, loc := range locs {
		file := filepath.Base(loc.File)
		po, ok := poos[file]
		if !ok {
			continue
		}

		// the owning element is the one whose line range contains loc.Line
		var elem *pool.SpecElement
		for _, s := range spansOf(file, po) {
			if loc.Line >= s.start && loc.Line <= s.end {
				elem = s.elem
				break
			}
		}
		if elem == nil {
			continue
		}

		issue := fmt.Sprintf("%v:%v: %v", loc.Line, loc.Col, loc.Msg)
		if node, ok := byElem[elem]; ok {
			node.Issues = append(node.Issues, issue)
			continue
		}
		byElem[elem] = &ErrorNode{Elem: elem, Issues: []string{issue}}
		order = append(order, elem)
	}

	nodes := make([]ErrorNode, 0, len(order))
	for _, elem := range order {
		nodes = append(nodes, *byElem[elem])
	}
	return nodes
}

// lineRangeOf returns the 1-based [start, end] line range that code occupies
// inside text. code is matched as a line-aligned block (preceded by the start of
// text or a newline, and followed by the end of text or a newline) so a fragment
// shared with another block is not matched by mistake. Returns (0, 0) if not found.
func lineRangeOf(text, code string) (int, int) {
	if code == "" {
		return 0, 0
	}
	for from := 0; from <= len(text); {
		idx := strings.Index(text[from:], code)
		if idx < 0 {
			return 0, 0
		}
		abs := from + idx
		end := abs + len(code)
		beforeOK := abs == 0 || text[abs-1] == '\n'
		afterOK := end == len(text) || text[end] == '\n'
		if beforeOK && afterOK {
			start := strings.Count(text[:abs], "\n") + 1
			return start, start + strings.Count(code, "\n")
		}
		from = abs + 1
	}
	return 0, 0
}

// ErrorLocation describes the file location an error message points to.
type ErrorLocation struct {
	File string
	Line int
	Col  int
	Msg  string
}

// catchErrorLocations parses compiler/extractor output and returns the file
// locations reported by each error message. Error lines are expected to follow
// the "file:line:col: message" format produced by ast.Pos.
func catchErrorLocations(buf *bytes.Buffer) []ErrorLocation {
	re := regexp.MustCompile(`(?m)^(.*?):(\d+):(\d+):\s*(.*)$`)
	matches := re.FindAllStringSubmatch(buf.String(), -1)

	var locs []ErrorLocation
	for _, m := range matches {
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		locs = append(locs, ErrorLocation{
			File: strings.TrimSpace(m[1]),
			Line: line,
			Col:  col,
			Msg:  strings.TrimSpace(m[4]),
		})
	}
	return locs
}

// recoverPools converts *.syz.txt.json files under specific directory to SpecPool, and save them
// into a map. The key of the map is file's path, and value is the SpecPool converts from key. Returns
// map and error messages.
func recoverPools(dir string) (map[string]*pool.SpecPool, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.syz.txt.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read json files under %v: %v", dir, err)
	}
	poos := make(map[string]*pool.SpecPool)
	for _, match := range matches {
		buf, err := os.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("failed to read %v: %v", match, err)
		}
		po, err := pool.NewSpecPoolFromJson(string(buf))
		if err != nil {
			return nil, fmt.Errorf("failed to convert %v to pool: %v", match, err)
		}
		poos[match] = po
	}
	return poos, nil
}

// createPrelPools creates preliminary pools of fidl syz specs. It hacks specs and
// temporarily patches them for successfully pool parsing. After converting to pool,
// specs is unhacked to keep consistence with original.
func createPrelPools(sysdir string, fidlsyzDir string, fixedDir string) error {
	// match fild syz specs
	fidlsyzs, err := filepath.Glob(filepath.Join(fidlsyzDir, "*.syz.txt"))
	if err != nil {
		return fmt.Errorf("faled to match fidlsyz under sysdir: %v", err)
	}

	// load nodes in sysdir
	global := make(map[string]bool)
	matches, err := filepath.Glob(filepath.Join(sysdir, "fuchsia", "*.txt"))
	if err != nil {
		return fmt.Errorf("failed to match specs under sysdir: %v", err)
	}
	for _, match := range matches {
		buf, err := os.ReadFile(match)
		if err != nil {
			return fmt.Errorf("failed to read %v: %v", match, err)
		}
		po, err := pool.NewSpecPoolFromSyzlang(string(buf))
		if err != nil {
			return fmt.Errorf("failed to convert %v to pool: %v", match, err)
		}
		for k := range *po {
			global[k] = true
		}
	}

	// traverse fidl syz specs, hack them, deduplicate nodes globally, convert to pools,
	// and unhack pools. Finally saved to fixedDir/*.syz.txt.json
	for _, fidlsyz := range fidlsyzs {
		buf, err := os.ReadFile(fidlsyz)
		if err != nil {
			return fmt.Errorf("failed to read %v: %v", fidlsyz, err)
		}
		// hack & convert to pool
		hackedSyzl := fidlsyzHacker.Hack(string(buf))
		po, err := pool.NewSpecPoolFromSyzlang(hackedSyzl)
		if err != nil {
			return fmt.Errorf("failed to convert hacked fild syz to pool: %v", err)
		}
		// deduplicate globally
		for k := range *po {
			if _, ok := global[k]; ok {
				delete(*po, k)
			} else {
				global[k] = true
			}
		}
		// unhack
		po = fidlsyzHacker.UnhackSpecPool(po)
		jstr, err := po.Json()
		if err != nil {
			return fmt.Errorf("failed to convert pool to json string: %v", err)
		}
		// save to *.syz.txt.json files
		filename := filepath.Base(fidlsyz)
		if err := os.WriteFile(filepath.Join(fixedDir, filename+".json"), []byte(jstr), 0644); err != nil {
			return fmt.Errorf("failed to write pool: %v", err)
		}
	}
	return nil
}

// setConfigs parse command-line flags and set programs configurations
func setConfigs() *ProgConfig {
	// comand-line flags
	var cfg ProgConfig
	flag.StringVar(&cfg.Model, "model", "", "the model to use")
	flag.StringVar(&cfg.Env, "env", filepath.Join(getcwd(), ".env"), "path to .env file")
	flag.StringVar(&cfg.Outdir, "outdir", "", "path to the output directory")
	flag.StringVar(&cfg.ExtractBin, "extract-bin", filepath.Join(getcwd(), "bin", "syz-extract"), "path to the syz-extract binary")
	flag.StringVar(&cfg.CheckBin, "check-bin", filepath.Join(getcwd(), "bin", "syz-check"), "path to the syz-check binary")
	flag.StringVar(&cfg.Kernel, "kernel", "", "path to the kernel used for spec generation")
	flag.BoolVar(&cfg.Resume, "resume", true, "whether to resume from previous interrupted run")
	flag.StringVar(&cfg.Sysdir, "sysdir", filepath.Join(getcwd(), "syzkaller", "sys"), "path to the sys directory (syzkaller/sys like structure)")
	flag.StringVar(&cfg.FidlsyzDir, "fidlsyz-dir", "", "path to the directory saved specs generated by fidlgen_syzkaller")
	flag.StringVar(
		&cfg.FixSysPrompt,
		"fix-sys-prompt",
		defaultFixSysPrompt(),
		"path to the fix system prompt file(s), use comma to sperate multiple files",
	)
	flag.IntVar(&cfg.MaxFix, "max-fix", 5, "maximum number of fix attempts for specs")
	flag.IntVar(&cfg.Jobs, "jobs", 1, "number of parallel jobs")
	flag.Parse()
	return &cfg
}
