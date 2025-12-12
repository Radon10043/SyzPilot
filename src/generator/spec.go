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
	"github.com/Radon10043/cloud/src/generator/check"
	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/Radon10043/cloud/src/generator/utils"
	"github.com/tmc/langchaingo/llms"
)

// writeSpec start prompting agent to outline todo tasks, generate specs, and fix specs for a global variable,
// return the final syzlang spec and whether it is valid
func writeSpec(
	kAgent *agent.Agent, sysPromptMap *map[string]string, gvEntry *database.GlobalVar, cfg *progConfig,
) (string, bool, error) {
	logger := log.New(os.Stdout, "["+gvEntry.Name+"][outline] ", log.LstdFlags|log.Lmsgprefix)
	outline, err := collectOutline(kAgent, (*sysPromptMap)["outline"], gvEntry, cfg, logger)
	if err != nil {
		return "", false, err
	}
	logger = log.New(os.Stdout, "["+gvEntry.Name+"][generate] ", log.LstdFlags|log.Lmsgprefix)
	jsonSpec, err := collectSpec(kAgent, (*sysPromptMap)["generate"], gvEntry, cfg, outline, logger)
	if err != nil {
		return "", false, err
	}
	logger = log.New(os.Stdout, "["+gvEntry.Name+"][fix] ", log.LstdFlags|log.Lmsgprefix)
	syzSpec, valid, err := fixSpec(kAgent, (*sysPromptMap)["fix"], gvEntry, cfg, jsonSpec, logger)
	if err != nil {
		return "", false, err
	}
	return syzSpec, valid, nil
}

// fixSpec start a loop to fix invalid syscall spec, also with the help of agent, return the final syzlang spec
// and whether it is valid
func fixSpec(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, cfg *progConfig, jsonSpec string, logger *log.Logger,
) (string, bool, error) {
	fdir := filepath.Join(cfg.Outdir, gvEntry.Name+"#"+cfg.Model)
	_ = os.MkdirAll(fdir, 0755)
	fp := filepath.Join(fdir, "spec.txt")

	// if spec.txt exists and resume is enabled, reuse existing spec
	var (
		spec  string
		found bool
	)
	if _, err := os.Stat(fp); err == nil && cfg.Resume {
		logger.Printf("Spec file exists, reuse existing.\n")
		buf, err := os.ReadFile(fp)
		if err != nil {
			return "", false, fmt.Errorf("failed to read file: %v\n", err)
		}
		spec = string(buf)
	} else { // otherwise, convert json spec to syzlang and write to spec.txt
		spec, err = utils.Json2syzlang(jsonSpec)
		if err != nil {
			return "", false, fmt.Errorf("failed to convert json spec to syzlang: %v\n", err)
		}
		if err = os.WriteFile(fp, []byte(spec), 0644); err != nil {
			return "", false, fmt.Errorf("failed to write spec file: %v\n", err)

		}
	}

	// make agent ready for fix loop
	kAgent.CleanMessages()
	if err := kAgent.AddSystemMessage(sysPrompt); err != nil {
		return "", false, fmt.Errorf("failed to add system prompt to agent: %v\n", err)
	}

	// check validity of spec and prompt agent to fix it if invalid
	buf, err := os.ReadFile(cfg.Prefix)
	if err != nil {
		return "", false, fmt.Errorf("failed to read prefix file: %v\n", err)
	}
	var (
		prefix string = string(buf)
		valid  bool   = false
		stdout *bytes.Buffer
		stderr *bytes.Buffer
	)
	for i := 0; i < cfg.MaxFix; i++ { // limit the number of fix attempts
		logger.Printf("Checking validity of spec ...\n")
		if spec == "" {
			return "", false, fmt.Errorf("empty spec, stop.\n")
		}
		// it's okay to ignore command error (last return value) here since it is not fatal
		stdout, stderr, valid, _ = checkSpecValidity(kAgent.ToolHelper.Sc, prefix+"\n\n"+spec)
		if valid {
			logger.Printf("Spec is valid!\n")
			break
		}
		logger.Printf(
			"Spec is invalid, trying to fix.\n"+
				"========== stdout ==========\n%s\n"+
				"========== stderr ==========\n%s\n",
			stdout.String(), stderr.String(),
		)
		specBlock := fmt.Sprintf("```syzlang\n%s\n```\n", spec)
		errBlock, err := createErrBlock(stdout, stderr, cfg)
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
			return "", false, fmt.Errorf("failed to extract syzlang code fence from fix response.\n")
		}
		err = os.WriteFile(fp, []byte(spec), 0644)
		if err != nil {
			return "", false, fmt.Errorf("failed to write spec file: %v\n", err)
		}
	}

	return spec, valid, nil
}

// checkSpecValidity check the validity of a syscall spec, return whether it is valid and error message if any
func checkSpecValidity(sc *check.SyzCheck, spec string) (*bytes.Buffer, *bytes.Buffer, bool, error) {
	err := sc.CleanWorkdir()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to clean syzkaller workdir: %v", err)
	}
	err = sc.AddSpec(spec)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to add spec to syzkaller workdir: %v", err)
	}
	stdout, stderr, cmdErr := sc.ExtractConst()
	if cmdErr != nil {
		return stdout, stderr, false, nil
	}
	stdout, stderr, cmdErr = sc.CheckValidity()
	if cmdErr != nil {
		return stdout, stderr, false, nil
	}
	return stdout, stderr, true, nil
}

// createErrBlock create an error block from stdout and stderr of `make extract` or `syz-check`
func createErrBlock(stdout *bytes.Buffer, stderr *bytes.Buffer, cfg *progConfig) (string, error) {
	// format stdout and stderr messages
	fmtStdout, err := formatMessages(stdout, cfg.Prefix)
	if err != nil {
		return "", fmt.Errorf("failed to extract error messages: %v\n", err)
	}
	fmtStderr, err := formatMessages(stderr, cfg.Prefix)
	if err != nil {
		return "", fmt.Errorf("failed to extract error messages: %v\n", err)
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
			return nil, fmt.Errorf("failed to run agent in fix loop: %v\n", err)
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
			return nil, fmt.Errorf("failed to execute tools in fix loop: %v\n", err)
		}
	}

	return response, nil
}

// collectSpec prompt agent to generate syscall spec or reuse existing spec for a global variable
func collectSpec(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, cfg *progConfig, outline string, logger *log.Logger,
) (string, error) {
	fdir := filepath.Join(cfg.Outdir, gvEntry.Name+"#"+cfg.Model)
	_ = os.MkdirAll(fdir, 0755)
	fp := filepath.Join(fdir, "spec.json")

	// if spec.json exist and resume is enabled, reuse existing spec
	if _, err := os.Stat(fp); err == nil && cfg.Resume {
		logger.Printf("Spec file exists, reuse existing.\n")
		buf, err := os.ReadFile(fp)
		if err != nil {
			return "", fmt.Errorf("failed to read spec file: %v\n", err)
		}
		outline = string(buf)
	}
	outlineMap := make(map[string]any)
	err := json.Unmarshal([]byte(outline), &outlineMap)
	if err != nil {
		return "", fmt.Errorf("failed to parse outline json: %v\n", err)
	}

	// prompt agent to generate spec iteratively until all todo tasks are done
	var found bool
	todoNum := len(outlineMap["todo"].([]any))
	for todoNum > 0 {
		kAgent.CleanMessages()
		response, err := genSpec(kAgent, sysPrompt, gvEntry, outline, logger)
		if err != nil {
			return "", err
		}
		outline, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "json")
		if !found {
			return "", fmt.Errorf("failed to extract json code fence from generate response.\n")
		}
		_, err = utils.Json2syzlang(outline)
		if err != nil {
			return "", fmt.Errorf("failed to generate valid json: %v\n", err)
		}
		err = os.WriteFile(fp, []byte(outline), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write spec file: %v\n", err)
		}
		err = json.Unmarshal([]byte(outline), &outlineMap)
		if err != nil {
			return "", fmt.Errorf("failed to parse outline json: %v\n", err)
		}
		todoNum = len(outlineMap["todo"].([]any))
	}
	logger.Printf("All todo tasks are done.\n")

	return outline, nil
}

// genSpec prompt agent to generate syscall spec iteratively
func genSpec(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, outline string, logger *log.Logger,
) (*llms.ContentResponse, error) {

	// make agent ready for spec generation stage
	var err error
	err = kAgent.AddSystemMessage(sysPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to add system prompt to agent: %v\n", err)
	}

	// prompt agent to generate syscall spec
	var response *llms.ContentResponse
	humanMsg := fmt.Sprintf("```c\n%s\n```\n\n```json\n%s```\n", gvEntry.Code, outline)
	kAgent.AddHumanMessage(humanMsg)
	for {
		logger.Printf("Query agent ...\n")
		response, err = kAgent.Query()
		if err != nil {
			return nil, fmt.Errorf("failed to run agent: %v\n", err)
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
			return nil, fmt.Errorf("failed to execute tools: %v\n", err)
		}
	}
	logger.Printf("Generation loop stop reason: %v", response.Choices[0].StopReason)

	return response, nil
}

// collectOutline prompt agent to outline todo tasks or reuse existing outline for a global variable
func collectOutline(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, cfg *progConfig, logger *log.Logger,
) (string, error) {
	fdir := filepath.Join(cfg.Outdir, gvEntry.Name+"#"+cfg.Model)
	_ = os.MkdirAll(fdir, 0755)
	fp := filepath.Join(fdir, "outline.json")

	// if outline.json doesn't exist or resume is disabled, prompt agent to outline todo tasks
	var (
		outline string
		found   bool
	)
	if _, err := os.Stat(fp); err != nil || !cfg.Resume {
		kAgent.CleanMessages()
		response, err := genOutline(kAgent, sysPrompt, gvEntry, logger)
		if err != nil {
			return "", err
		}
		outline, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "json")
		if !found {
			return "", fmt.Errorf("failed to extract json code fence from outline response.\n")
		}
		_, err = utils.Json2syzlang(outline)
		if err != nil {
			return "", fmt.Errorf("failed to generate valid json: %v\n", err)
		}
		err = os.WriteFile(fp, []byte(outline), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write outline file: %v\n", err)
		}
	} else {
		logger.Printf("Outline file exists, reuse existing and skip outline stage.\n")
		data, err := os.ReadFile(fp)
		if err != nil {
			return "", fmt.Errorf("failed to read outline file: %v\n", err)
		}
		outline = string(data)
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
		return nil, fmt.Errorf("failed to add system prompt to agent: %v\n", err)
	}

	// prompt agent to outline todo tasks
	var response *llms.ContentResponse
	kAgent.AddHumanMessage(gvEntry.Code)
	for {
		response, err = kAgent.Query()
		if err != nil {
			return nil, fmt.Errorf("failed to run agent in outline stage: %v\n", err)
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
			return nil, fmt.Errorf("failed to execute tools in outline stage: %v\n", err)
		}
	}

	return response, nil
}
