import asyncio
import time
from restack_ai import Restack


async def main():
    client = Restack()

    workflow_id = f"{int(time.time() * 1000)}-ManagerAgent"
    run_id = await client.schedule_workflow(
        workflow_name="ManagerAgent", workflow_id=workflow_id
    )

    result = await client.get_workflow_result(workflow_id=workflow_id, run_id=run_id)
    print(result)

    exit(0)


def run_schedule_workflow():
    asyncio.run(main())


if __name__ == "__main__":
    run_schedule_workflow()
