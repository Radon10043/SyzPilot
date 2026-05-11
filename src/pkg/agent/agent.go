package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/Radon10043/SyzPilot/src/pkg/database"
	myTools "github.com/Radon10043/SyzPilot/src/pkg/tools"
	"github.com/Radon10043/SyzPilot/src/pkg/utils"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Agent struct {
	Ctx          context.Context             // context for llm operations
	Model        *openai.LLM                 // model instance
	Messages     []llms.MessageContent       // message history
	Temperature  float32                     // temperature for llm
	MaxTokens    int                         // max tokens for single response of llm
	ToolMap      map[string]myTools.ToolExec // available tools for the agent
	ToolHelper   *myTools.ToolHelper         // helper for tool execution
	ToolCallHist []llms.ToolCall             // history of tool calls, used for checking repetition and whether agent is stucked
}

// Purge purges the message history and tool call history of the agent
func (a *Agent) Purge() {
	a.Messages = []llms.MessageContent{}
	a.ToolCallHist = []llms.ToolCall{}
}

// AddSystemMessage appends a system message to the agent's message history
func (a *Agent) AddSystemMessage(input string) error {
	if len(a.Messages) != 0 {
		return fmt.Errorf("system message can only be added to empty message history")
	}
	msg := llms.TextParts(llms.ChatMessageTypeSystem, input)
	a.Messages = append(a.Messages, msg)
	return nil
}

// AddHumanMessage appends a human message to the agent's message history
func (a *Agent) AddHumanMessage(input string) {
	msg := llms.TextParts(llms.ChatMessageTypeHuman, input)
	a.Messages = append(a.Messages, msg)
}

// SaveMessageFile saves the current message history to a file
func (a *Agent) SaveMessageFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	a.SaveMessage(f)
	return nil
}

// SaveMessage saves the current message history to a given file handle
func (a *Agent) SaveMessage(f *os.File) {
	for _, msg := range a.Messages {
		fmt.Fprintf(f, "========== ROLE: %v ==========\n", msg.Role)
		for _, part := range msg.Parts {
			str := utils.Part2string(part)
			fmt.Fprintf(f, "%v\n", str)
		}
	}
}

// Query query the llm with the current messages and update the history
func (a *Agent) Query() (*llms.ContentResponse, error) {
	// query the llm with existing messages
	avaTools := []llms.Tool{}
	for _, te := range a.ToolMap {
		avaTools = append(avaTools, te.Tool)
	}
	response, err := a.Model.GenerateContent(
		a.Ctx,
		a.Messages,
		llms.WithTools(avaTools),
		llms.WithTemperature(float64(a.Temperature)),
		llms.WithMaxTokens(a.MaxTokens),
	)
	if err != nil {
		return nil, err
	}
	// update the message history with ai's response
	choice := response.Choices[0]
	aiResponse := llms.TextParts(llms.ChatMessageTypeAI, choice.Content)
	for _, tc := range choice.ToolCalls {
		aiResponse.Parts = append(aiResponse.Parts, tc)
	}
	a.Messages = append(a.Messages, aiResponse)
	return response, nil
}

// QueryLoop query the llm in a loop until there is no tool call in the response,
// return the final response and error if any
func (a *Agent) QueryLoop() (*llms.ContentResponse, error) {
	var response *llms.ContentResponse
	var err error
	for {
		response, err = a.Query()
		if err != nil {
			return nil, err
		}
		if len(response.Choices[0].ToolCalls) == 0 {
			break
		}
		err = a.ExecTools()
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

// ExecTools execute the tools called by the llm and update the message history
func (a *Agent) ExecTools() error {
	lastMsg := a.Messages[len(a.Messages)-1]
	for _, part := range lastMsg.Parts {
		// skip non-tool-call parts
		tc, ok := part.(llms.ToolCall)
		if !ok {
			continue
		}
		// check if the agent is repeating same tool calling, if so, we consider it is stucked, we need
		// to stop it to avoid token wasting
		if a.repeatSameTool() {
			return fmt.Errorf("agent is repeating same tool calling, stop execution to avoid infinite loop")
		}
		// check if the agent has called tools more than a certain number of times. If it has, the agent may be stucked,
		// we also need to stop it to avoid token wasting
		if len(a.ToolCallHist) >= 25 {
			return fmt.Errorf("agent has called tools for %d times, stop execution to avoid infinite loop", len(a.ToolCallHist))
		}
		// execute the tool based on its name
		tcResp := llms.MessageContent{}
		err := error(nil)
		toolExec, ok := a.ToolMap[tc.FunctionCall.Name]
		if !ok {
			err = fmt.Errorf("no executor found for tool: %s", tc.FunctionCall.Name)
		} else {
			tcResp, err = toolExec.Exec(&tc, a.ToolHelper)
		}
		if err != nil {
			return err
		}
		a.Messages = append(a.Messages, tcResp)
		a.ToolCallHist = append(a.ToolCallHist, tc)
	}
	return nil
}

func (a *Agent) repeatSameTool() bool {
	n := len(a.ToolCallHist)
	if n < 1 {
		return false
	}

	// if the last five tool call have same arguments, we consider agent is repeating
	// same tool calling
	var (
		obj1      map[string]any
		repCnt    int = 1
		maxTolRep int = 5 // maximum tolerance for repetition
	)
	json.Unmarshal([]byte(a.ToolCallHist[n-1].FunctionCall.Arguments), &obj1)
	for i := n - 2; i >= max(n-maxTolRep, 0); i-- {
		if a.ToolCallHist[i].FunctionCall.Name != a.ToolCallHist[n-1].FunctionCall.Name {
			return false
		}
		var obj2 map[string]any
		json.Unmarshal([]byte(a.ToolCallHist[i].FunctionCall.Arguments), &obj2)
		if reflect.DeepEqual(obj1, obj2) {
			repCnt++
		}
		if repCnt >= maxTolRep {
			break
		}
	}
	return repCnt >= maxTolRep
}

// NewAgent creates a new agent with the given database and initializes the tool map
func NewAgent(db *database.Database, llm *openai.LLM) *Agent {
	// use all tools
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
	return &Agent{
		Ctx:         context.Background(),
		Model:       llm,
		Messages:    []llms.MessageContent{},
		Temperature: 0.2,
		MaxTokens:   128 << 10, // 128k
		ToolMap:     toolMap,
		ToolHelper:  toolHelper,
	}
}

// TODO: let's refactor these NewAgent funcs to NewAgent(opts...)
// NewAgentWithTools creates a new agent with the given database and customized tool map
func NewAgentWithTools(db *database.Database, llm *openai.LLM, toolMap map[string]myTools.ToolExec) *Agent {
	toolHelper := &myTools.ToolHelper{
		Db: db,
	}
	return &Agent{
		Ctx:         context.Background(),
		Model:       llm,
		Messages:    []llms.MessageContent{},
		Temperature: 0.2,
		MaxTokens:   128 << 10, // 128k
		ToolMap:     toolMap,
		ToolHelper:  toolHelper,
	}
}
