package stage

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/pool"
	"github.com/Radon10043/cloud/src/pkg/queue"
	"github.com/Radon10043/cloud/src/pkg/utils"
	"github.com/tmc/langchaingo/llms"
)

// ExecGenerateStep execute the generate step of the write spec process
func ExecGenerateStep(
	kAgent *agent.Agent, sysPrompt string, gvEntry *database.GlobalVar, logger *log.Logger, sh *StageHelper,
) error {
	type genContent struct {
		Spec     []pool.SpecElement    `json:"spec"`
		Required []queue.TaskQueueElem `json:"required"`
	}
	var (
		jstr    string
		err     error
		gc      genContent
		telem   *queue.TaskQueueElem
		ctxPool *pool.SpecPool
		ctxSpec strings.Builder
	)

	// get the top element from Tqueue
	telem, err = sh.Tqueue.Pop()
	if err != nil {
		return fmt.Errorf("failed to pop from Tqueue: %v", err)
	}

	// if the element is already in SyzPool, skip generate and determine whether to reuse it
	// according its type and the type of element in SyzPool. Specifically, if the element
	// is syscall, skip generate and do not reuse; otherwise reuse. Note that if the element
	// is init_syscall but element in SyzPool is syscall, we should set its type to init_syscall.
	if sh.SyzPool.Exists(telem.Name) {
		se, err := sh.SyzPool.Get(telem.Name)
		if err != nil {
			return fmt.Errorf("failed to get element from SyzPool: %v", err)
		}
		if telem.Type == queue.TaskHeapElemTypeSyscall.String() {
			logger.Printf("Element %v (%v) already in SyzPool, skip generate and do not reuse\n", telem.Name, telem.Type)
			return nil
		}
		logger.Printf("Element %v (%v) already in SyzPool, skip generate and reuse it\n", telem.Name, telem.Type)
		if se.Type == queue.TaskHeapElemTypeSyscall.String() && telem.Type == queue.TaskHeapElemTypeInitSyscall.String() {
			se.Type = queue.TaskHeapElemTypeInitSyscall.String()
		}
		sh.Rpool.Insert(se)
		return nil
	}

	// if the element is already in Pool, skip generate and reuse it
	if sh.Pool.Exists(telem.Name) {
		logger.Printf("Element %v (%v) already in Pool, skip generate and reuse Pool's element\n", telem.Name, telem.Type)
		se, err := sh.Pool.Get(telem.Name)
		if err != nil {
			return fmt.Errorf("failed to get element from Pool: %v", err)
		}
		sh.Spool.Insert(se)
		return nil
	}

	// if the element is already in Spool, dont generate again
	if sh.Spool.Exists(telem.Name) {
		logger.Printf("Element %v (%v) already in Spool, skip generate\n", telem.Name, telem.Type)
		return nil
	}

	// extract consistency reference element from sh.Pool and sh.Rpool, merge them with Spool,
	// so that agent is likely to generate consistent specifications
	ctxPool = getConsPool(sh.Pool)
	ctxPool.Merge(getConsPool(sh.Rpool))
	ctxSpec.WriteString(ctxPool.Syzlang())

	// prompt agent to generate spec for the element
	if jstr, err = collectSpec(kAgent, sysPrompt, ctxSpec, gvEntry, telem, logger); err != nil {
		return err
	}
	if err = sh.SaveQueryMessages(kAgent, "generate-"); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(jstr), &gc); err != nil {
		return err
	}

	// process generated spec and required tasks
	for _, se := range gc.Spec {
		sh.Spool.Insert(se)
	}
	for _, te := range gc.Required {
		sh.Tqueue.Push(&te)
	}
	return nil
}

// collectSpec prompt agent to generate syscall spec for a given task element
func collectSpec(
	kAgent *agent.Agent, sysPrompt string, ctxSpec strings.Builder, gvEntry *database.GlobalVar, telem *queue.TaskQueueElem, logger *log.Logger,
) (string, error) {
	// prompt agent to generate spec to complete part of todo tasks
	var (
		found bool
		jstr  string
	)
	kAgent.CleanMessages()
	response, err := genSpec(kAgent, sysPrompt, ctxSpec, gvEntry, telem, logger)
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
	kAgent *agent.Agent, sysPrompt string, ctxSpec strings.Builder, gvEntry *database.GlobalVar, telem *queue.TaskQueueElem, logger *log.Logger,
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
		"```c\n%s\n```\n\nPlease write specification for %s `%s`\n\n",
		gvEntry.Code, telem.Type, telem.Name,
	)
	if ctxSpec.Len() > 0 {
		humanMsg += fmt.Sprintf(
			"The following specifications have been generated so far:\n```syzlang\n%s\n```\n\n",
			ctxSpec.String(),
		)
	}
	kAgent.AddHumanMessage(humanMsg)
	logger.Printf("Query agent (%s %s) ...\n", telem.Type, telem.Name)
	for {
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
