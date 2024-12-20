import asyncio
import os
from src.functions.function import manager_agent_prompt, calculator_agent_prompt, runtime_tool
from src.client import client
from src.workflows.workflow import ManagerAgent, CalculatorAgent
from watchfiles import run_process


async def main():
    await client.start_service(
        workflows=[ManagerAgent, CalculatorAgent],
        functions=[manager_agent_prompt, calculator_agent_prompt, runtime_tool],
    )


def run_services():
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("Service interrupted by user. Exiting gracefully.")


def watch_services():
    watch_path = os.getcwd()
    print(f"Watching {watch_path} and its subdirectories for changes...")
    run_process(watch_path, recursive=True, target=run_services)
