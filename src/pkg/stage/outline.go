package stage

import (
	"fmt"
	"log"

	"github.com/Radon10043/SyzPilot/src/pkg/agent"
	"github.com/Radon10043/SyzPilot/src/pkg/database"
	"github.com/Radon10043/SyzPilot/src/pkg/queue"
	"github.com/Radon10043/SyzPilot/src/pkg/utils"
	"github.com/tmc/langchaingo/llms"
)

// ExecOutlineStep execute the outline step of the write spec process
func ExecOutlineStep(
	kAgent *agent.Agent, sysPrompt string, entry database.Entry, logger *log.Logger, sh *StageHelper,
) error {
	var (
		err  error
		jstr string
	)
	if jstr, err = collectOutline(kAgent, sysPrompt, entry, logger); err != nil {
		return err
	}
	if sh.Tqueue, err = queue.NewTaskQueueFromJson(jstr); err != nil {
		return fmt.Errorf("failed to create spec task queue from outline: %v", err)
	}
	if err = sh.SaveQueryMessages(kAgent, "outline-"); err != nil {
		return err
	}
	return nil
}

// collectOutline prompt agent to outline todo tasks or reuse existing outline for a database entry
func collectOutline(
	kAgent *agent.Agent, sysPrompt string, entry database.Entry, logger *log.Logger,
) (string, error) {
	var (
		outline string
		found   bool
	)
	kAgent.Purge()
	response, err := genOutline(kAgent, sysPrompt, entry, logger)
	if err != nil {
		return "", err
	}
	outline, found = utils.ExtractFirstCodeBlock(response.Choices[0].Content, "json")
	if !found {
		return "", fmt.Errorf("failed to extract json code fence from outline response")
	}
	return outline, nil
}

// genOutline prompt agent to outline todo tasks for a database entry
func genOutline(
	kAgent *agent.Agent, sysPrompt string, entry database.Entry, logger *log.Logger,
) (*llms.ContentResponse, error) {
	// make agent ready for outline stage
	var err error
	err = kAgent.AddSystemMessage(sysPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to add system prompt to agent: %v", err)
	}

	// prompt agent to outline todo tasks
	var response *llms.ContentResponse
	kAgent.AddHumanMessage(entry.GetCode())
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
