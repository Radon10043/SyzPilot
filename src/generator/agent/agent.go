package agent

import (
	"context"
	"fmt"
	"os"

	myTools "github.com/Radon10043/cloud/src/generator/tools"
	"github.com/Radon10043/cloud/src/generator/utils"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Agent struct {
	SystemPrompt string                // system prompt for the agent
	Ctx          context.Context       // context for llm operations
	Model        *openai.LLM           // model instance
	Tools        []llms.Tool           // available tools
	Messages     []llms.MessageContent // message history
	Temperature  float32               // temperature for llm
}

// CleanMessages clear the message history of the agent
func (a *Agent) CleanMessages() {
	a.Messages = []llms.MessageContent{}
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

// SaveMessages saves the current message history to a file
func (a *Agent) SaveMessages(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, msg := range a.Messages {
		fmt.Fprintf(f, "========== ROLE: %v ==========\n", msg.Role)
		for _, part := range msg.Parts {
			str := utils.PartToString(part)
			fmt.Fprintf(f, "%v\n", str)
		}
	}
	return nil
}

// Query query the llm with the current messages and update the history
func (a *Agent) Query() (*llms.ContentResponse, error) {
	// query the llm with existing messages
	response, err := a.Model.GenerateContent(a.Ctx, a.Messages, llms.WithTools(a.Tools), llms.WithTemperature(float64(a.Temperature)))
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
		// execute the tool based on its name
		tcResp := llms.MessageContent{}
		err := error(nil)
		switch tc.FunctionCall.Name {
		case "get_current_weather":
			tcResp, err = myTools.ExecGetCurrentWeather(tc)
		case "get_func_code_by_name":
			tcResp, err = myTools.ExecGetFuncCodeByName(tc)
		case "get_enum_code_by_enumerator":
			tcResp, err = myTools.ExecGetEnumCodeByEnumerator(tc)
		case "get_struct_code_by_name":
			tcResp, err = myTools.ExecGetStructCodeByName(tc)
		case "get_union_code_by_name":
			tcResp, err = myTools.ExecGetUnionCodeByName(tc)
		case "get_global_var_code_by_name":
			tcResp, err = myTools.ExecGetGlobalVarCodeByName(tc)
		case "get_typedef_type_by_define":
			tcResp, err = myTools.ExecGetTypedefTypeByDefine(tc)
		case "get_typedef_code_by_define":
			tcResp, err = myTools.ExecGetTypedefCodeByDefine(tc)
		case "get_macro_def_code_by_name":
			tcResp, err = myTools.ExecGetMacroDefCodeByName(tc)
		case "check_spec_validity":
			tcResp, err = myTools.ExecCheckSpecValidity(tc)
		default:
			err = fmt.Errorf("unknown tool: %s", tc.FunctionCall.Name)
		}
		if err != nil {
			return err
		}
		a.Messages = append(a.Messages, tcResp)
	}
	return nil
}
