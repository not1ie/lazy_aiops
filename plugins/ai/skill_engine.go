package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SkillResult 执行技能返回结果
type SkillResult struct {
	Reply     string        `json:"reply"`
	ToolCalls []AIToolTrace `json:"tool_calls"`
}

// RunSkill 运行指定技能
func (s *AIService) RunSkill(skill *AISkill, parameters map[string]interface{}) (*SkillResult, error) {
	aiCfg := s.core.AI.SnapshotConfig()
	if aiCfg.Model == "" {
		return nil, fmt.Errorf("AI 提供商配置未激活")
	}

	// 1. 构建提示词
	prompt := skill.SystemPrompt
	if len(parameters) > 0 {
		paramStr, _ := json.MarshalIndent(parameters, "", "  ")
		prompt += fmt.Sprintf("\n\n当前输入参数:\n%s", string(paramStr))
	}

	// 2. 解析绑定的工具
	var boundToolNames []string
	if skill.ToolBindings != "" {
		_ = json.Unmarshal([]byte(skill.ToolBindings), &boundToolNames)
	}

	// 3. 构建可用的 Tool 列表
	var availableTools []aiToolDefinition
	if len(boundToolNames) > 0 {
		allTools := s.availableToolsForScope("general") // 获取所有可能的工具
		for _, tool := range allTools {
			for _, boundName := range boundToolNames {
				if tool.Name == boundName {
					availableTools = append(availableTools, tool)
					break
				}
			}
		}
	}

	// 4. 执行逻辑
	result := &SkillResult{}

	// 构建虚拟请求以便复用内部规划逻辑
	req := &ChatRequest{
		Message: prompt, // 把 System Prompt 作为首次输入
	}

	// 4.1 如果没有绑定任何工具，直接调用大模型
	if len(availableTools) == 0 {
		reply, err := s.callAI(prompt)
		if err != nil {
			return nil, err
		}
		result.Reply = reply
		return result, nil
	}

	// 4.2 如果绑定了工具，让模型自行决定调用
	// 这里复用原本 chat.go 中的规划逻辑：s.planToolCalls
	var history []ChatMessage // 空历史
	plan, err := s.planToolCalls(req, nil, history, availableTools)
	if err != nil {
		// 规划失败回退
		reply, err := s.callAI(prompt)
		if err != nil {
			return nil, err
		}
		result.Reply = reply
		return result, nil
	}

	if !plan.UseTools || len(plan.ToolCalls) == 0 {
		// 模型决定不用工具
		reply, err := s.callAI(prompt)
		if err != nil {
			return nil, err
		}
		result.Reply = reply
		return result, nil
	}

	// 执行工具调用
	var toolContexts []string
	var traces []AIToolTrace

	for _, call := range plan.ToolCalls {
		// 查找对应工具
		var targetTool *aiToolDefinition
		for _, t := range availableTools {
			if t.Name == call.Name {
				targetTool = &t
				break
			}
		}

		if targetTool != nil {
			// 执行工具
			execResult, err := targetTool.Run(call.Arguments)
			trace := AIToolTrace{
				Name:      call.Name,
				Reason:    call.Reason,
				Arguments: call.Arguments,
			}
			if err != nil {
				trace.Status = "error"
				trace.Summary = fmt.Sprintf("执行失败: %v", err)
				toolContexts = append(toolContexts, fmt.Sprintf("- [%s] 调用失败: %v", call.Name, err))
			} else {
				trace.Status = "success"
				trace.Summary = "执行成功"
				toolContexts = append(toolContexts, fmt.Sprintf("- [%s] 调用成功，结果: %s", call.Name, truncate(execResult, 2000)))
			}
			traces = append(traces, trace)
		}
	}

	result.ToolCalls = traces

	// 结合工具结果做最终总结
	finalPrompt := fmt.Sprintf("%s\n\n您执行了一些工具，以下是执行结果:\n%s\n\n请根据以上结果，完成最终的回答。", prompt, strings.Join(toolContexts, "\n"))
	finalReply, err := s.callAI(finalPrompt)
	if err != nil {
		return nil, err
	}
	result.Reply = finalReply

	return result, nil
}
