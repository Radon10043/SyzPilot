package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/ast"
	"github.com/Radon10043/cloud/src/pkg/check"
	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/pool"
	"github.com/Radon10043/cloud/src/pkg/queue"
	"github.com/Radon10043/cloud/src/pkg/utils"
	"github.com/tmc/langchaingo/llms"
)

// writeSpecHelper is a helper struct to hold intermediate results during spec writing
type writeSpecHelper struct {
	Workdir    string // path to work directory
	Next       string // next step
	SpecPrefix string // spec prefix
	LogPrefix  string // log prefix for logging

	Tqueue     *queue.TaskQueue // priority queue for tracking elements to generate
	TqueuePath string           // path to Tqueue json file

	Spool     *pool.SpecPool // pool for tracking elements to be fixed
	SpoolPath string         // path to Spool json file

	Pool     *pool.SpecPool // pool for tracking already completed (maybe unfixed) elements
	PoolPath string         // path to pool json file
}

// WriteTqueue write the Tqueue field to Tqueue file
func (wsh *writeSpecHelper) WriteTqueue() error {
	data, err := wsh.Tqueue.Json()
	if err != nil {
		return fmt.Errorf("failed to convert Tqueue to json: %v", err)
	}
	if err = os.WriteFile(wsh.TqueuePath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write Tqueue file: %v", err)
	}
	return nil
}

// WriteSpool write the Spool field to Spool file
func (wsh *writeSpecHelper) WriteSpool() error {
	data, err := wsh.Spool.Json()
	if err != nil {
		return fmt.Errorf("failed to convert Spool to json: %v", err)
	}
	if err = os.WriteFile(wsh.SpoolPath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write Spool file: %v", err)
	}
	return nil
}

// WritePool write the Pool field to Pool file
func (wsh *writeSpecHelper) WritePool() error {
	data, err := wsh.Pool.Json()
	if err != nil {
		return fmt.Errorf("failed to convert Pool to json: %v", err)
	}
	if err = os.WriteFile(wsh.PoolPath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write Pool file: %v", err)
	}
	return nil
}

// WriteCurrStat write the current state of Tqueue, Spool, and Pool to files
func (wsh *writeSpecHelper) WriteCurrStat() error {
	// write current state of Tqueue, Spool, and Pool to files after each step
	if err := wsh.WriteTqueue(); err != nil {
		return fmt.Errorf("failed to write Tqueue: %v", err)
	}
	if err := wsh.WriteSpool(); err != nil {
		return fmt.Errorf("failed to write Spool: %v", err)
	}
	if err := wsh.WritePool(); err != nil {
		return fmt.Errorf("failed to write Pool: %v", err)
	}
	return nil
}

// RecoverProgress recover existing progress from workdir
func (wsh *writeSpecHelper) RecoverProgress() error {
	// recover Tqueue field
	if _, err := os.Stat(wsh.TqueuePath); err == nil {
		data, err := os.ReadFile(wsh.TqueuePath)
		if err != nil {
			return fmt.Errorf("failed to read Tqueue file: %v", err)
		}
		wsh.Tqueue, err = queue.NewTaskQueueFromJson(string(data))
		if err != nil {
			return fmt.Errorf("failed to recover Tqueue from json: %v", err)
		}
	}

	// recover Spool field
	if _, err := os.Stat(wsh.SpoolPath); err == nil {
		data, err := os.ReadFile(wsh.SpoolPath)
		if err != nil {
			return fmt.Errorf("failed to read Spool file: %v", err)
		}
		if err = json.Unmarshal(data, &wsh.Spool); err != nil {
			return fmt.Errorf("failed to unmarshal Spool json: %v", err)
		}
	}

	// recover Pool field
	if _, err := os.Stat(wsh.PoolPath); err == nil {
		data, err := os.ReadFile(wsh.PoolPath)
		if err != nil {
			return fmt.Errorf("failed to read Pool file: %v", err)
		}
		if err = json.Unmarshal(data, &wsh.Pool); err != nil {
			return fmt.Errorf("failed to unmarshal Pool json: %v", err)
		}
	}

	return nil
}

// SaveQueryMessages save the current messages of kAgent to a temp file with given prefix
func (wsh *writeSpecHelper) SaveQueryMessages(kAgent *agent.Agent, prefix string) error {
	timestamp := time.Now().UnixMilli()
	fn := fmt.Sprintf("%s%d.msg", prefix, timestamp)
	fp := filepath.Join(wsh.Workdir, fn)
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf("failed to create file for saving messages: %v", err)
	}
	defer f.Close()
	kAgent.SaveMessage(f)
	return nil
}

// UpdateNextStep update the Next field according to the current state of writeSpecHelper
func (wsh *writeSpecHelper) UpdateNextStep() {
	wsh.Next = wsh.NextStep()
}

// NextStep return the next step to execute according to the current state of writeSpecHelper.
func (wsh *writeSpecHelper) NextStep() string {
	if wsh.ShouldOutline() {
		return "outline"
	}
	if wsh.ShouldComplete() {
		return "complete"
	}
	if wsh.ShouldFix() {
		return "fix"
	}
	// TODO: not sure if there are any omissions
	return "generate"
}

// ShouldOutline return whether the outline step should be executed, which is true
// only when Tqueue is nil, i.e. no progress has been made
func (wsh *writeSpecHelper) ShouldOutline() bool {
	return wsh.Tqueue == nil
}

// ShouldComplete return whether the complete step should be executed, which is true
// when:
//   - both Tqueue and Spool are empty; or
//   - all elements in Tqueue and Spool are already in Pool
func (wsh *writeSpecHelper) ShouldComplete() bool {
	if wsh.Tqueue.Empty() && wsh.Spool.Empty() {
		return true
	}
	allInPool := true
	for _, te := range wsh.Tqueue.Slice() {
		if !wsh.Pool.Exists(te.Name) {
			allInPool = false
			break
		}
	}
	if allInPool {
		for _, se := range *wsh.Spool {
			if !wsh.Pool.Exists(se.Name) {
				allInPool = false
				break
			}
		}
	}
	return allInPool
}

// ShouldFix return whether the fix step should be executed, which is true when:
//   - Tqueue is empty and Spool is not empty; or
//   - syscall in Spool and top element in Tqueue is syscall
func (wsh *writeSpecHelper) ShouldFix() bool {
	if wsh.Tqueue.Empty() && !wsh.Spool.Empty() {
		return true
	}
	hasSyscall := false
	for _, se := range *wsh.Spool {
		if se.Type == queue.TaskHeapElemTypeSyscall.String() {
			hasSyscall = true
		}
	}
	topElem, err := wsh.Tqueue.Peek()
	return hasSyscall && (wsh.Tqueue.Empty() || (err == nil && topElem.Type == queue.TaskHeapElemTypeSyscall.String()))
}

// UpdatePool update the Pool field with elements from given SpecPool
func (wsh *writeSpecHelper) UpdatePool(sq *pool.SpecPool) {
	for _, se := range *sq {
		// if se.Name not in wsh.Pool, add it to the pool directly
		if !wsh.Pool.Exists(se.Name) {
			wsh.Pool.Insert(*se)
			continue
		}
		// if se.Name already in wsh.Pool, update it only if se is valid and wsh.Pool[se.Name] is invalid
		pse, err := wsh.Pool.Get(se.Name)
		if err != nil {
			return
		}
		if !pse.Valid && se.Valid {
			wsh.Pool.Insert(*se)
		}
	}
}

// writeSpec start prompting agent to outline todo tasks, generate specs, and fix specs for a global variable,
// return the final syzlang spec and whether it is valid
func writeSpec(
	kAgent *agent.Agent, sysPromptMap *map[string]string, gvEntry *database.GlobalVar, cfg *ProgConfig, specPrefix string, logPrefix string,
) (string, error) {
	// init and pool default value for writeSpecHelper
	specdir := filepath.Join(cfg.Outdir, "specs", gvEntry.Name+"#"+cfg.Model)
	var wsh *writeSpecHelper = &writeSpecHelper{
		Workdir:    filepath.Join(specdir),
		Next:       "outline",
		SpecPrefix: specPrefix,
		LogPrefix:  logPrefix,
		Tqueue:     nil,
		TqueuePath: filepath.Join(specdir, ".tqueue"),
		Spool:      &pool.SpecPool{},
		SpoolPath:  filepath.Join(specdir, ".spool"),
		Pool:       &pool.SpecPool{},
		PoolPath:   filepath.Join(specdir, ".pool"),
	}
	err := os.MkdirAll(wsh.Workdir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create workdir: %v", err)
	}

	// if resume is enabled, recover existing progress
	if cfg.Resume {
		if err = wsh.RecoverProgress(); err != nil {
			return "", fmt.Errorf("failed to recover progress: %v", err)
		}
	} // otherwise start from scratch

	// start the write spec loop, limit the number of iterations to avoid infinite loop
	for range 100 {
		wsh.UpdateNextStep()
		if wsh.Next == "complete" {
			wsh.Tqueue.Clear()
			wsh.Spool.Clear()
			break
		}
		if err := execWriteStep(kAgent, sysPromptMap, gvEntry, cfg, wsh); err != nil {
			return "", fmt.Errorf("failed to exec write step: %v", err)
		}
		if err := wsh.WriteCurrStat(); err != nil {
			return "", fmt.Errorf("failed to write current state: %v", err)
		}
	}
	if err = wsh.WriteCurrStat(); err != nil {
		return "", fmt.Errorf("failed to write current state: %v", err)
	}

	return wsh.Pool.Syzlang(pool.WithValidComment(true)), nil
}

// execWriteStep execute one step of the write spec process according to wsh.Next
func execWriteStep(
	kAgent *agent.Agent, sysPromptMap *map[string]string, gvEntry *database.GlobalVar, cfg *ProgConfig, wsh *writeSpecHelper,
) error {
	logger := log.New(os.Stdout, wsh.LogPrefix+"["+gvEntry.Name+"]["+wsh.Next+"] ", log.LstdFlags|log.Lmsgprefix)
	switch wsh.Next {
	case "outline":
		return execOutlineStep(kAgent, (*sysPromptMap)["outline"], gvEntry, logger, wsh)
	case "generate":
		return execGenerateStep(kAgent, (*sysPromptMap)["generate"], gvEntry, logger, wsh)
	case "fix":
		return execFixStep(kAgent, (*sysPromptMap)["fix"], cfg, logger, wsh)
	case "complete":
	default:
		return fmt.Errorf("unknown next step: %s", wsh.Next)
	}
	return nil
}

// execFixStep execute the fix step
func execFixStep(kAgent *agent.Agent, sysPrompt string, cfg *ProgConfig, logger *log.Logger, wsh *writeSpecHelper) error {
	var (
		ospec  string         // original syzlang spec before fixing
		nspec  string         // new syzlang spec after fixing
		nspool *pool.SpecPool // new SpecPool after fixing
		valid  bool           // whether the final full spec is valid
		err    error
	)

	// if all elements is valid, skip fixing
	allValid := true
	for _, se := range *wsh.Spool {
		if !se.Valid {
			allValid = false
			break
		}
	}
	if allValid {
		logger.Printf("All elements in Spool are valid, skip fixing\n")
		wsh.Spool.Clear()
		return nil
	}

	ospec = wsh.Spool.Syzlang()
	if nspec, valid, err = fixSpec(kAgent, sysPrompt, cfg, ospec, logger, wsh); err != nil {
		return err
	}
	if err = wsh.SaveQueryMessages(kAgent, "fix-"); err != nil {
		return err
	}

	// if new spec is valid, assigned it to wsh.Spool
	if valid {
		nspool, err = ast.Syzlang2SpecPool(nspec)
		if err != nil {
			return err
		}
		nspool.SyncType(wsh.Spool)
		for _, v := range *nspool {
			v.Valid = true
		}
		wsh.Spool = nspool
	} // otherwise new spec is invalid, reuse old Spool

	wsh.UpdatePool(wsh.Spool)
	wsh.Spool.Clear()
	return nil

}

// fixSpec start a loop to fix invalid syscall spec, also with the help of agent, return the final syzlang spec
// and whether it is valid
func fixSpec(
	kAgent *agent.Agent, sysPrompt string, cfg *ProgConfig, spec string, logger *log.Logger, wsh *writeSpecHelper,
) (string, bool, error) {
	// make agent ready for fix loop
	kAgent.CleanMessages()
	if err := kAgent.AddSystemMessage(sysPrompt); err != nil {
		return "", false, fmt.Errorf("failed to add system prompt to agent: %v", err)
	}

	// check validity of spec and prompt agent to fix it if invalid
	var (
		valid  bool = false
		stdout *bytes.Buffer
		stderr *bytes.Buffer
		found  bool
	)
	for i := 0; i < cfg.MaxFix; i++ { // limit the number of fix attempts
		logger.Printf("Checking validity of spec ...\n")
		if spec == "" {
			return "", false, fmt.Errorf("empty spec, stop")
		}
		// it's okay to ignore command error (last return value) here since it is not fatal
		stdout, stderr, valid, _ = checkSpecValidity(kAgent.ToolHelper.Sc, wsh.SpecPrefix+"\n\n"+spec)
		if valid {
			logger.Printf("Spec is valid!\n")
			break
		}
		logger.Printf("Spec is invalid, trying to fix. stdout=%q stderr=%q\n", stdout.String(), stderr.String())
		specBlock := fmt.Sprintf("```syzlang\n%s\n```\n", spec)
		errBlock, err := createErrBlock(stdout, stderr, wsh)
		if err != nil {
			return "", false, err
		}
		kAgent.AddHumanMessage(specBlock + "\n" + errBlock)
		response, err := fixSpecLoop(kAgent, logger)
		if err != nil {
			return "", false, err
		}
		spec, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "syzlang")
		if !found {
			return "", false, fmt.Errorf("failed to extract syzlang code fence from fix response")
		}
	}

	return spec, valid, nil
}

// checkSpecValidity check the validity of a syscall spec, return whether it is valid and error message if any
func checkSpecValidity(sc *check.SpecCheck, spec string) (*bytes.Buffer, *bytes.Buffer, bool, error) {
	err := sc.RestoreWorkdir()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to clean syzkaller workdir: %v", err)
	}
	fpath, err := sc.AddSpec(spec)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to add spec to syzkaller workdir: %v", err)
	}
	stdout, stderr, valid := sc.ExtractConst(filepath.Base(fpath))
	if !valid {
		return stdout, stderr, valid, nil
	}
	stdout, stderr, valid = sc.CheckValidity()
	if !valid {
		return stdout, stderr, valid, nil
	}
	return stdout, stderr, valid, nil
}

// createErrBlock create an error block from stdout and stderr of `make extract` or `syz-check`
func createErrBlock(stdout *bytes.Buffer, stderr *bytes.Buffer, wsh *writeSpecHelper) (string, error) {
	// format stdout and stderr messages
	fmtStdout, err := formatMessages(stdout, wsh.SpecPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to extract error messages: %v", err)
	}
	fmtStderr, err := formatMessages(stderr, wsh.SpecPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to extract error messages: %v", err)
	}

	// format error message block, mainly adjust line numbers according to prefix length
	var errBlock strings.Builder
	errBlock.WriteString("```\n")
	for _, msg := range fmtStdout {
		errBlock.WriteString(msg + "\n")
	}
	for _, msg := range fmtStderr {
		errBlock.WriteString(msg + "\n")
	}
	errBlock.WriteString("```\n")

	return errBlock.String(), nil
}

// formatMessages format error messages from stdout/stderr of `make extract` or `syz-check`, specifically
// adjust line numbers according to prefix length
func formatMessages(buf *bytes.Buffer, prefix string) ([]string, error) {
	type specError struct {
		Path   string
		Line   int
		Column int
		Issue  string
	}

	// a helper function to split a raw error message into path, line number, column number, and issue
	splitHelper := func(str string) *specError {
		tmp := strings.Split(str, ":")
		// number of elements should greater than 4, specifically:
		// 0: file path; 1: line number; 2: column number; 3-n: error message
		if len(tmp) < 4 {
			return nil
		}
		path := tmp[0]
		issue := strings.Join(tmp[3:], ":")
		line, err := strconv.Atoi(tmp[1])
		if err != nil {
			return nil
		}
		column, err := strconv.Atoi(tmp[2])
		if err != nil {
			return nil
		}
		return &specError{
			Path:   path,
			Line:   line,
			Column: column,
			Issue:  issue,
		}
	}

	// adjust line numbers in error messages
	prefixLines := strings.Split(prefix, "\n")
	prefixLineLen := len(prefixLines)
	var msgs []string
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		se := splitHelper(line)
		if se == nil {
			msgs = append(msgs, line)
		} else {
			se.Line -= prefixLineLen
			msgs = append(msgs, fmt.Sprintf("%s:%d:%d:%s", se.Path, se.Line, se.Column, se.Issue))
		}
	}

	return msgs, scanner.Err()
}

// fixSpecLoop run a loop to fix invalid syscall spec
func fixSpecLoop(kAgent *agent.Agent, logger *log.Logger) (*llms.ContentResponse, error) {
	// prompt agent to fix spec
	var (
		response *llms.ContentResponse
		err      error
	)
	for {
		response, err = kAgent.Query()
		if err != nil {
			return nil, fmt.Errorf("failed to run agent in fix loop: %v", err)
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
			return nil, fmt.Errorf("failed to execute tools in fix loop: %v", err)
		}
	}

	return response, nil
}

// execGenerateStep execute the generate step of the write spec process
func execGenerateStep(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, logger *log.Logger, wsh *writeSpecHelper,
) error {
	type genContent struct {
		Spec     []pool.SpecElement    `json:"spec"`
		Required []queue.TaskQueueElem `json:"required"`
	}
	var (
		jstr      string
		err       error
		gc        genContent
		telem     *queue.TaskQueueElem
		spoolSpec strings.Builder
	)

	// get the top element from Tqueue
	telem, err = wsh.Tqueue.Pop()
	if err != nil {
		return fmt.Errorf("failed to pop from Tqueue: %v", err)
	}

	// if the element is already in Pool, skip generate and reuse it
	if wsh.Pool.Exists(telem.Name) {
		logger.Printf("Element %v already in pool, skip generate and reuse pool's element\n", telem.Name)
		se, err := wsh.Pool.Get(telem.Name)
		if err != nil {
			return fmt.Errorf("failed to get element from Pool: %v", err)
		}
		wsh.Spool.Insert(se)
		return nil
	}

	// if the element is already in Spool, dont generate again
	if wsh.Spool.Exists(telem.Name) {
		logger.Printf("Element %v already in spool, skip generate\n", telem.Name)
		return nil
	}

	// put all resource and init_syscall elements from Pool, and present all Spool's elements in
	// prompts so that we can ensure spec as consistent as possible
	for _, se := range *wsh.Pool {
		if se.Type == "resource" || se.Type == "init_syscall" {
			wsh.Spool.Insert(*se)
		}
	}
	if !wsh.Spool.Empty() {
		spoolSpec.WriteString(wsh.Spool.Syzlang())
	}

	// prompt agent to generate spec for the element
	if jstr, err = collectSpec(kAgent, sysPrompt, spoolSpec, gvEntry, telem, logger); err != nil {
		return err
	}
	if err = wsh.SaveQueryMessages(kAgent, "generate-"); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(jstr), &gc); err != nil {
		return err
	}

	// process generated spec and required tasks
	for _, se := range gc.Spec {
		wsh.Spool.Insert(se)
	}
	for _, te := range gc.Required {
		wsh.Tqueue.Push(&te)
	}
	return nil
}

// collectSpec prompt agent to generate syscall spec for a given task element
func collectSpec(
	kAgent *agent.Agent, sysPrompt string, spoolSpec strings.Builder, gvEntry *database.GlobalVar, telem *queue.TaskQueueElem, logger *log.Logger,
) (string, error) {
	// prompt agent to generate spec to complete part of todo tasks
	var (
		found bool
		jstr  string
	)
	kAgent.CleanMessages()
	response, err := genSpec(kAgent, sysPrompt, spoolSpec, gvEntry, telem, logger)
	if err != nil {
		return "", err
	}
	jstr, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "json")
	if !found {
		return "", fmt.Errorf("failed to extract json code fence from generate response")
	}
	return jstr, nil
}

// genSpec prompt agent to generate syscall spec iteratively
func genSpec(
	kAgent *agent.Agent, sysPrompt string, spoolSpec strings.Builder, gvEntry *database.GlobalVar, telem *queue.TaskQueueElem, logger *log.Logger,
) (*llms.ContentResponse, error) {
	// make agent ready for spec generation stage
	var err error
	err = kAgent.AddSystemMessage(sysPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to add system prompt to agent: %v", err)
	}

	// prompt agent to generate syscall spec
	var response *llms.ContentResponse
	humanMsg := fmt.Sprintf(
		"```c\n%s\n```\n\nPlease write specification for %s `%s`\n",
		gvEntry.Code, telem.Type, telem.Name,
	)
	if spoolSpec.Len() > 0 {
		humanMsg += fmt.Sprintf(
			"The following specifications have been generated so far:\n```syzlang\n%s\n```\n",
			spoolSpec.String(),
		)
	}
	kAgent.AddHumanMessage(humanMsg)
	for {
		logger.Printf("Query agent ...\n")
		response, err = kAgent.Query()
		if err != nil {
			return nil, fmt.Errorf("failed to run agent: %v", err)
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
			return nil, fmt.Errorf("failed to execute tools: %v", err)
		}
	}
	logger.Printf("Generation loop stop reason: %v", response.Choices[0].StopReason)

	return response, nil
}

// execOutlineStep execute the outline step of the write spec process
func execOutlineStep(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, logger *log.Logger, wsh *writeSpecHelper,
) error {
	var (
		err  error
		jstr string
	)
	if jstr, err = collectOutline(kAgent, sysPrompt, gvEntry, logger); err != nil {
		return err
	}
	if wsh.Tqueue, err = queue.NewTaskQueueFromJson(jstr); err != nil {
		return fmt.Errorf("failed to create spec task queue from outline: %v", err)
	}
	if err = wsh.SaveQueryMessages(kAgent, "outline-"); err != nil {
		return err
	}
	return nil
}

// collectOutline prompt agent to outline todo tasks or reuse existing outline for a global variable
func collectOutline(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, logger *log.Logger,
) (string, error) {
	var (
		outline string
		found   bool
	)
	kAgent.CleanMessages()
	response, err := genOutline(kAgent, sysPrompt, gvEntry, logger)
	if err != nil {
		return "", err
	}
	outline, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "json")
	if !found {
		return "", fmt.Errorf("failed to extract json code fence from outline response")
	}
	return outline, nil
}

// genOutline prompt agent to outline todo tasks for a global variable
func genOutline(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, logger *log.Logger,
) (*llms.ContentResponse, error) {
	// make agent ready for outline stage
	var err error
	err = kAgent.AddSystemMessage(sysPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to add system prompt to agent: %v", err)
	}

	// prompt agent to outline todo tasks
	var response *llms.ContentResponse
	kAgent.AddHumanMessage(gvEntry.Code)
	for {
		response, err = kAgent.Query()
		if err != nil {
			return nil, fmt.Errorf("failed to run agent in outline stage: %v", err)
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
			return nil, fmt.Errorf("failed to execute tools in outline stage: %v", err)
		}
	}

	return response, nil
}
