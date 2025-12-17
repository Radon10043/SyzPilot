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

	"github.com/Radon10043/cloud/src/generator/agent"
	"github.com/Radon10043/cloud/src/generator/ast"
	"github.com/Radon10043/cloud/src/generator/check"
	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/Radon10043/cloud/src/generator/utils"
	"github.com/tmc/langchaingo/llms"
)

// writeSpecHelper is a helper struct to hold intermediate results during spec writing
type writeSpecHelper struct {
	Outline     string // outline json string
	OutlinePath string // path to outline json file
	Jstr        string // spec json string
	JstrPath    string // path to spec json file
	Spec        string // syzlang spec string
	SpecPath    string // path to syzlang spec file
	Valid       bool   // whether the spec is valid
	Workdir     string // path to work directory
	Next        string // next step
	DotNext     string // path to .next file, sync with Next field
	SpecPrefix  string // spec prefix
	LogPrefix   string // log prefix for logging
}

// WriteOutline write the outline field to outline file
func (wsh *writeSpecHelper) WriteOutline() error {
	if err := os.WriteFile(wsh.OutlinePath, []byte(wsh.Outline), 0644); err != nil {
		return fmt.Errorf("failed to write outline file: %v", err)
	}
	return nil
}

// WriteJstr write the jstr field to spec json file
func (wsh *writeSpecHelper) WriteJstr() error {
	if err := os.WriteFile(wsh.JstrPath, []byte(wsh.Jstr), 0644); err != nil {
		return fmt.Errorf("failed to write spec json file: %v", err)
	}
	return nil
}

// WriteSpec write the spec field to syzlang spec file
func (wsh *writeSpecHelper) WriteSpec() error {
	if err := os.WriteFile(wsh.SpecPath, []byte(wsh.Spec), 0644); err != nil {
		return fmt.Errorf("failed to write spec file: %v", err)
	}
	return nil
}

// WriteNext update the .next file and next field with the next step
func (wsh *writeSpecHelper) WriteNext(next string) error {
	if err := os.WriteFile(wsh.DotNext, []byte(next), 0644); err != nil {
		return fmt.Errorf("failed to update .next file: %v", err)
	}
	wsh.Next = next
	return nil
}

// RecoverProgress recover existing progress from workdir
func (wsh *writeSpecHelper) RecoverProgress() error {
	// recover outline field
	if _, err := os.Stat(wsh.OutlinePath); err == nil {
		data, err := os.ReadFile(wsh.OutlinePath)
		if err != nil {
			return fmt.Errorf("failed to read outline file: %v", err)
		}
		wsh.Outline = string(data)
	}

	// recover jstr field
	if _, err := os.Stat(wsh.JstrPath); err == nil {
		data, err := os.ReadFile(wsh.JstrPath)
		if err != nil {
			return fmt.Errorf("failed to read spec json file: %v", err)
		}
		wsh.Jstr = string(data)
	}

	// recover spec field
	if _, err := os.Stat(wsh.SpecPath); err == nil {
		data, err := os.ReadFile(wsh.SpecPath)
		if err != nil {
			return fmt.Errorf("failed to read spec file: %v", err)
		}
		wsh.Spec = string(data)
	}

	// recover next field
	if _, err := os.Stat(wsh.DotNext); err == nil {
		data, err := os.ReadFile(wsh.DotNext)
		if err != nil {
			return fmt.Errorf("failed to read .next file: %v", err)
		}
		wsh.Next = strings.TrimSpace(string(data))
	}

	return nil
}

// SaveQueryMessages save the current messages of kAgent to a temp file with given prefix
func (wsh *writeSpecHelper) SaveQueryMessages(kAgent *agent.Agent, prefix string) error {
	msgf, err := os.CreateTemp(wsh.Workdir, prefix+"*.msg")
	if err != nil {
		return fmt.Errorf("failed to create temp file for saving messages: %v", err)
	}
	defer msgf.Close()
	kAgent.SaveMessage(msgf)
	return nil
}

// writeSpec start prompting agent to outline todo tasks, generate specs, and fix specs for a global variable,
// return the final syzlang spec and whether it is valid
func writeSpec(
	kAgent *agent.Agent, sysPromptMap *map[string]string, gvEntry *database.GlobalVar, cfg *ProgConfig, specPrefix string, logPrefix string,
) (string, bool, error) {
	// init and set default value for writeSpecHelper
	specdir := filepath.Join(cfg.Outdir, "specs", gvEntry.Name+"#"+cfg.Model)
	var wsh *writeSpecHelper = &writeSpecHelper{
		OutlinePath: filepath.Join(specdir, "outline.json"),
		JstrPath:    filepath.Join(specdir, "spec.json"),
		SpecPath:    filepath.Join(specdir, "spec.txt"),
		Workdir:     filepath.Join(specdir),
		Next:        "outline",
		DotNext:     filepath.Join(specdir, ".next"),
		SpecPrefix:  specPrefix,
		LogPrefix:   logPrefix,
	}
	err := os.MkdirAll(wsh.Workdir, 0755)
	if err != nil {
		return "", false, fmt.Errorf("failed to create workdir: %v", err)
	}

	// if resume is enabled, recover existing progress
	if cfg.Resume {
		if err = wsh.RecoverProgress(); err != nil {
			return "", false, fmt.Errorf("failed to recover progress: %v", err)
		}
	} else { // otherwise start from scratch
		if err = wsh.WriteNext("outline"); err != nil {
			return "", false, fmt.Errorf("failed to write next step: %v", err)
		}
	}

	// start the write spec loop, limit the number of iterations to avoid infinite loop
	for range 100 {
		if err := execWriteStep(kAgent, sysPromptMap, gvEntry, cfg, wsh); err != nil {
			return "", false, fmt.Errorf("failed to exec write step: %v", err)
		}
		if wsh.Next == "complete" {
			break
		}
	}
	return wsh.Spec, wsh.Valid, nil
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
		jspec *ast.JsonSpec
		err   error
	)
	if wsh.Spec, wsh.Valid, err = fixSpec(kAgent, sysPrompt, cfg, wsh.Spec, logger, wsh); err != nil {
		return err
	}
	if err = wsh.SaveQueryMessages(kAgent, "fix-"); err != nil {
		return err
	}
	if err = wsh.WriteSpec(); err != nil {
		return err
	}
	// Update wsh.Jstr according to the validity of wsh.Spec
	if wsh.Valid {
		if wsh.Jstr, err = ast.Syzlang2json(wsh.Spec); err != nil {
			return fmt.Errorf("failed to convert valid spec to json: %v", err)
		}
		if err = wsh.WriteJstr(); err != nil {
			return err
		}
		jspec, err := ast.Syzlang2JsonSpec(wsh.Spec)
		if err != nil {
			return fmt.Errorf("failed to convert valid spec to json: %v", err)
		}
		if len(jspec.Todo) > 0 {
			return wsh.WriteNext("generate")
		}
	} else { // Invalid spec cannot be converted to json/JsonSpec, reuse latest json string
		err = json.Unmarshal([]byte(wsh.Jstr), &jspec)
		if err != nil {
			return fmt.Errorf("failed to parse existing spec json: %v", err)
		}
		if len(jspec.Todo) > 0 {
			return wsh.WriteNext("generate")
		}
	}
	return wsh.WriteNext("complete")

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
	err = sc.AddSpec(spec)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to add spec to syzkaller workdir: %v", err)
	}
	stdout, stderr, valid := sc.ExtractConst()
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

// execOutlineStep execute the outline step
func execGenerateStep(kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, logger *log.Logger, wsh *writeSpecHelper) error {
	var err error
	if wsh.Jstr, err = collectSpec(kAgent, sysPrompt, gvEntry, wsh.Jstr, logger); err != nil {
		return err
	}
	if err = wsh.SaveQueryMessages(kAgent, "generate-"); err != nil {
		return err
	}
	if err = wsh.WriteJstr(); err != nil {
		return err
	}
	if wsh.Spec, err = ast.Json2syzlang(wsh.Jstr); err != nil {
		return fmt.Errorf("failed to convert spec json to syzlang: %v", err)
	}
	if err = wsh.WriteSpec(); err != nil {
		return err
	}
	return wsh.WriteNext("fix")
}

// collectSpec prompt agent to generate syscall spec or reuse existing spec for a global variable
func collectSpec(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, outline string, logger *log.Logger,
) (string, error) {
	// prompt agent to generate spec to complete part of todo tasks
	var (
		found bool
		jstr  string
	)
	kAgent.CleanMessages()
	response, err := genSpec(kAgent, sysPrompt, gvEntry, outline, logger)
	if err != nil {
		return "", err
	}
	jstr, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "json")
	if !found {
		return "", fmt.Errorf("failed to extract json code fence from generate response")
	}
	_, err = ast.Json2syzlang(jstr)
	if err != nil {
		return "", fmt.Errorf("failed to generate valid json: %v", err)
	}
	return jstr, nil
}

// genSpec prompt agent to generate syscall spec iteratively
func genSpec(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, outline string, logger *log.Logger,
) (*llms.ContentResponse, error) {
	// make agent ready for spec generation stage
	var err error
	err = kAgent.AddSystemMessage(sysPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to add system prompt to agent: %v", err)
	}

	// prompt agent to generate syscall spec
	var response *llms.ContentResponse
	humanMsg := fmt.Sprintf("```c\n%s\n```\n\n```json\n%s```\n", gvEntry.Code, outline)
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
func execOutlineStep(kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, logger *log.Logger, wsh *writeSpecHelper) error {
	var err error
	if wsh.Outline, err = collectOutline(kAgent, sysPrompt, gvEntry, logger); err != nil {
		return err
	}
	if err = wsh.SaveQueryMessages(kAgent, "outline-"); err != nil {
		return err
	}
	if err = wsh.WriteOutline(); err != nil {
		return err
	}
	wsh.Jstr = wsh.Outline
	if err = wsh.WriteJstr(); err != nil {
		return err
	}
	return wsh.WriteNext("generate")
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
	_, err = ast.Json2syzlang(outline)
	if err != nil {
		return "", fmt.Errorf("failed to generate valid json: %v", err)
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
