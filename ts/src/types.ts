import { ChatCompletionTool } from "openai/resources"

export type Chain = {
  id: number
  messages: string
  created_at: string
  parent_function_call_id: number | null
  status:
  | "awaiting_llm_processing"
  | "llm_processing"
  | "stop_awaiting_user"
  | "awaiting_function_call"
}

export type Agent = {
  id?: number
  system_prompt: string
  name: string
  tools: ChatCompletionTool[]
  delegation_tool?: {
    description?: string
  }
}

export const agent_to_tool = (agent: Agent): ChatCompletionTool => {
  return {
    type: "function",
    function: {
      name: `delegate_to_${agent.name}`,
      description: agent.delegation_tool?.description || `delegate to ${agent.name}`,
      parameters: {
        type: "object",
        properties: {
          message: {
            type: "string",
            description:
              "the full message to send to the agent including all supporting context and details",
          },
        },
      },
    },
  }
}