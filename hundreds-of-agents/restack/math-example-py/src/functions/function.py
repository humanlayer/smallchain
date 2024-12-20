from typing import Any, Iterable
from restack_ai.function import function, log
from openai import OpenAI
from dataclasses import dataclass
import os
from dotenv import load_dotenv
from openai.types.chat import (
    ChatCompletionToolParam,
    ChatCompletionMessageParam,
    ChatCompletionMessage,
)

load_dotenv()


def calc_agent_tools() -> Iterable[ChatCompletionToolParam]:
    return [
        {
            "type": "function",
            "function": {
                "name": "add",
                "description": "Add two numbers",
                "parameters": {
                    "type": "object",
                    "properties": {"a": {"type": "number"}, "b": {"type": "number"}},
                    "required": ["a", "b"],
                },
            },
        },
        {
            "type": "function",
            "function": {
                "name": "subtract",
                "description": "Subtract two numbers",
                "parameters": {
                    "type": "object",
                    "properties": {"a": {"type": "number"}, "b": {"type": "number"}},
                    "required": ["a", "b"],
                },
            },
        },
        {
            "type": "function",
            "function": {
                "name": "multiply",
                "description": "Multiply two numbers",
                "parameters": {
                    "type": "object",
                    "properties": {"a": {"type": "number"}, "b": {"type": "number"}},
                    "required": ["a", "b"],
                },
            },
        },
        {
            "type": "function",
            "function": {
                "name": "divide",
                "description": "Divide two numbers",
                "parameters": {
                    "type": "object",
                    "properties": {"a": {"type": "number"}, "b": {"type": "number"}},
                    "required": ["a", "b"],
                },
            },
        },
    ]


def manager_agent_tools() -> Iterable[ChatCompletionToolParam]:
    return [
        {
            "type": "function",
            "function": {
                "name": "delegate_to_CalculatorAgent",
                "description": "delegate a task to the calculator agent",
                "parameters": {
                    "type": "object",
                    "properties": {"message": {"type": "string"}},
                    "required": ["message"],
                },
            },
        }
    ]


@function.defn()
async def run_tool(tool_name: str, tool_kwargs: dict) -> str:
    try:
        log.info(
            "run_tool function started", tool_name=tool_name, tool_kwargs=tool_kwargs
        )
        return f"Tool {tool_name} completed with result {tool_kwargs}"
    except Exception as e:
        log.error("run_tool function failed", error=e)
        raise e


@dataclass
class ManagerAgentInput:
    message: str


@function.defn()
async def manager_agent_prompt(
    input: ManagerAgentInput,
) -> list[ChatCompletionMessageParam | ChatCompletionMessage]:
    try:
        log.info("manager_agent_prompt function started", input=input)
        client = OpenAI(api_key=os.environ.get("OPENAI_API_KEY"))

        messages: list[ChatCompletionMessageParam] = []
        messages.append(
            {
                "role": "system",
                "content": "you are a project manager skilled at orchestration and delegation",
            }
        )
        messages.append({"role": "user", "content": input.message})

        response = client.chat.completions.create(
            model="gpt-4o",
            messages=messages,
            tools=manager_agent_tools(),
        )
        log.info("manager_agent_prompt function completed", response=response)
        return messages + [response.choices[0].message]
    except Exception as e:
        log.error("manager_agent_prompt function failed", error=e)
        raise e


@dataclass
class ManagerAgentContinueInput:
    messages: list[ChatCompletionMessageParam | ChatCompletionMessage]


@function.defn()
async def manager_agent_continue(
    input: ManagerAgentContinueInput,
) -> list[ChatCompletionMessageParam | ChatCompletionMessage]:
    try:
        log.info("manager_agent_continue function started", input=input)
        client = OpenAI(api_key=os.environ.get("OPENAI_API_KEY"))

        response = client.chat.completions.create(
            model="gpt-4o",
            messages=input.messages,  # type: ignore  #fuck
            tools=manager_agent_tools(),
        )
        log.info("manager_agent_continue function completed", response=response)
        return input.messages + [response.choices[0].message]
    except Exception as e:
        log.error("manager_agent_continue function failed", error=e)
        raise e


@dataclass
class DelegatedAgentInput:
    message: str


@function.defn()
async def calculator_agent_prompt(
    input: DelegatedAgentInput,
) -> list[ChatCompletionMessageParam | ChatCompletionMessage]:
    try:
        log.info("calculator_agent_prompt function started", input=input)
        client = OpenAI(api_key=os.environ.get("OPENAI_API_KEY"))

        messages: list[ChatCompletionMessageParam] = []
        messages.append(
            {"role": "system", "content": "you are a skilled calculator agent"}
        )
        messages.append({"role": "user", "content": input.message})

        response = client.chat.completions.create(
            model="gpt-4o",  # probably can use mini but lets be consistent with the parent example
            messages=messages,
            tools=calc_agent_tools(),
        )
        log.info("calculator_agent_prompt function completed", response=response)
        return messages + [response.choices[0].message]
    except Exception as e:
        log.error("calculator_agent_prompt function failed", error=e)
        raise e


@dataclass
class ToolInput:
    name: str
    args: dict


runtime_tools = {
    "add": lambda a, b: a + b,
    "subtract": lambda a, b: a - b,
    "multiply": lambda a, b: a * b,
    "divide": lambda a, b: a / b,
}


@function.defn()
async def runtime_tool(input: ToolInput) -> float:
    return float(runtime_tools[input.name](**input.args))
