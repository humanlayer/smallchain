from datetime import timedelta
import json
from restack_ai.workflow import workflow, import_functions, log
from openai.types.chat import ChatCompletionMessageParam, ChatCompletionMessage

with import_functions():
    from src.functions.function import manager_agent_prompt
    from src.functions.function import calculator_agent_prompt
    from src.functions.function import manager_agent_continue
    from src.functions.function import runtime_tool

INPUT_QUERY = "add (2 + 5), then add (3 + 4), then add the results"


@workflow.defn()
class ManagerAgent:
    @workflow.run
    async def run(self) -> str:
        log.info("ManagerWorkflow started")
        messages: list[
            ChatCompletionMessageParam | ChatCompletionMessage
        ] = await workflow.step(
            manager_agent_prompt,
            input={"message": INPUT_QUERY},
            start_to_close_timeout=timedelta(seconds=120),
        )
        result: ChatCompletionMessage = messages[-1]  # type: ignore
        log.info("step result", result=result)

        if result.tool_calls:
            for tool_call in result.tool_calls:
                tool_call_id = tool_call.id
                tool_call_function = tool_call.function
                tool_call_function_args = json.loads(tool_call_function.arguments)
                assert "message" in tool_call_function_args, "message is required"

                if tool_call_function.name.startswith("delegate_to_"):
                    delegate_agent_name = tool_call_function.name.split("delegate_to_")[
                        1
                    ]
                    workflow_id = f"{tool_call_id}-{delegate_agent_name}"
                    delegate_agent_result = await workflow.child_execute(
                        delegate_agent_name,
                        workflow_id,
                        input=tool_call_function_args,  # should be {message: str}
                    )
                    messages.append(
                        {
                            "role": "tool",
                            "content": json.dumps(delegate_agent_result),
                            "tool_call_id": tool_call_id,
                        }
                    )
                else:
                    # hard code this thing, for now assume all the manager's tools are delegation tools
                    log.error("Unknown tool call, skipping", tool_call=tool_call)

            await workflow.step(
                manager_agent_continue,
                input={"messages": messages},
                start_to_close_timeout=timedelta(seconds=120),
            )

        log.info("ManagerWorkflow completed", result=result)
        return messages[-1].content  # type: ignore


@workflow.defn()
class CalculatorAgent:
    @workflow.run
    async def run(self):
        log.info("CalculatorWorkflow started")
        result = await workflow.step(
            calculator_agent_prompt,
            input={"message": INPUT_QUERY},
            start_to_close_timeout=timedelta(seconds=120),
        )
        if result.tool_calls:
            for tool_call in result.tool_calls:
                tool_call_id = tool_call.id
                tool_call_function = tool_call.function
                tool_call_function_name = tool_call_function.name
                tool_call_function_args = json.loads(
                    tool_call_function.arguments
                )  # should be {message: str}

                if tool_call_function_name.startswith("delegate_to_"):
                    # hard code assume no delegation, maybe these two can be combined
                    log.error("Unknown tool call, skipping", tool_call=tool_call)
                else:
                    await workflow.step(
                        runtime_tool,
                        input={
                            "name": tool_call_function_name,
                            "args": tool_call_function_args,
                        },
                    )
                    messages.append(
                        {
                            "role": "tool",
                            "content": tool_call_function_args["message"],
                            "tool_call_id": tool_call_id,
                        }
                    )

            await workflow.step(
                manager_agent_continue,
                input={"messages": messages},
                start_to_close_timeout=timedelta(seconds=120),
            )

        log.info("CalculatorWorkflow completed", result=result)
        return result
