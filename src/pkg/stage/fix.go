package stage

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/check"
	"github.com/Radon10043/cloud/src/pkg/pool"
	"github.com/Radon10043/cloud/src/pkg/queue"
	"github.com/Radon10043/cloud/src/pkg/utils"
	"github.com/tmc/langchaingo/llms"
)

// ExecFixStep execute the fix step
func ExecFixStep(kAgent *agent.Agent, sysPrompt string, logger *log.Logger, sh *StageHelper) error {
	var (
		bfPool *pool.SpecPool // pool for elements before fixing
		bfSpec string         // syzlang spec for elements before fixing
		afPool *pool.SpecPool // new SpecPool after fixing
		afSpec string         // new syzlang spec after fixing
		valid  bool           // whether the final full spec is valid
		err    error
	)

	// if all elements is valid, skip fixing
	allValid := true
	for _, se := range *sh.Spool {
		if !se.Valid {
			allValid = false
			break
		}
	}
	if allValid {
		logger.Printf("All elements in Spool are valid, skip fixing\n")
		sh.Spool.Clear()
		return nil
	}

	// sh.Spool may not have init_syscall and syscall at the same time. The cause should be agent
	// generate false init_syscall and syscalls is right but they are already in SyzPool. For such
	// have init_syscall but do not have syscall case, we directly return.
	hasInitSyscall := false
	hasSyscall := false
	for _, se := range *sh.Spool {
		if se.Type == queue.TaskHeapElemTypeInitSyscall.String() {
			hasInitSyscall = true
		}
		if se.Type == queue.TaskHeapElemTypeSyscall.String() {
			hasSyscall = true
		}
	}
	if hasInitSyscall && !hasSyscall {
		logger.Printf("Spool has init_syscall but do not have syscall, skip fixing\n")
		sh.UpdatePool(sh.Spool)
		sh.Spool.Clear()
		return nil
	}

	bfPool = getConsPool(sh.Pool)
	bfPool.Merge(getConsPool(sh.Rpool))
	bfPool.Merge(sh.Spool)
	bfSpec = bfPool.Syzlang()
	if afSpec, valid, err = fixSpec(kAgent, sysPrompt, bfSpec, logger, sh); err != nil {
		return err
	}
	if err = sh.SaveQueryMessages(kAgent, "fix-"); err != nil {
		return err
	}

	// if new spec is valid, assigned it to sh.Spool
	if valid {
		afPool, err = pool.NewSpecPoolFromSyzlang(afSpec)
		if err != nil {
			return err
		}
		afPool.SyncType(sh.Spool)
		for _, v := range *afPool {
			v.Valid = true
		}
		sh.Spool = afPool
	} // otherwise new spec is invalid, reuse old Spool
	sh.UpdatePool(sh.Spool)
	sh.Spool.Clear()

	return nil

}

// fixSpec start a loop to fix invalid syscall spec, also with the help of agent, return the final syzlang spec
// and whether it is valid
func fixSpec(
	kAgent *agent.Agent, sysPrompt string, spec string, logger *log.Logger, sh *StageHelper,
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
	for i := 0; i < sh.MaxFix; i++ { // limit the number of fix attempts
		logger.Printf("Checking validity of spec ...\n")
		if spec == "" {
			return "", false, fmt.Errorf("empty spec, stop")
		}
		// it's okay to ignore command error (last return value) here since it is not fatal
		stdout, stderr, valid, _ = checkSpecValidity(sh.Scheck, sh.SpecPrefix+"\n\n"+spec)
		if valid {
			logger.Printf("Spec is valid!\n")
			return spec, valid, nil
		}
		logger.Printf("Spec is invalid, trying to fix. stdout=%q stderr=%q\n", stdout.String(), stderr.String())
		specBlock := fmt.Sprintf("```syzlang\n%s\n```\n", spec)
		errBlock, err := createErrBlock(stdout, stderr, sh)
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
	// final check of spec validity
	stdout, stderr, valid, _ = checkSpecValidity(sh.Scheck, sh.SpecPrefix+"\n\n"+spec)
	if valid {
		logger.Printf("Spec is valid!\n")
	} else {
		logger.Printf("Failed to fix spec after max attempts. Final stdout=%q stderr=%q\n", stdout.String(), stderr.String())
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
func createErrBlock(stdout *bytes.Buffer, stderr *bytes.Buffer, sh *StageHelper) (string, error) {
	// format stdout and stderr messages
	fmtStdout, err := formatMessages(stdout, sh.SpecPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to extract error messages: %v", err)
	}
	fmtStderr, err := formatMessages(stderr, sh.SpecPrefix)
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
