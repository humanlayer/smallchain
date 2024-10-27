import OpenAI from "openai"
import { ChatCompletionTool } from "openai/resources"
import { open } from "sqlite"
import sqlite3 from "sqlite3"
import { Agent, agent_to_tool, Chain } from "./types"


export const logger = {
  info: (...message: any) => {
    console.log(`${new Date().toISOString()} INFO ${message}`)
  },
  error: (...message: any) => {
    console.log(`${new Date().toISOString()} ERROR ${message}`)
  },
  debug: (...message: any) => {
    console.log(`${new Date().toISOString()} DEBUG ${message}`)
  },
}

async function add({ x, y }: { x: number; y: number }): Promise<number> {
  return x + y
}

const add_tools = (): ChatCompletionTool[] => [
  {
    type: "function",
    function: {
      name: "add",
      description: "add two numbers",
      parameters: {
        type: "object",
        properties: {
          x: {
            type: "number",
            description: "The first number to add",
          },
          y: {
            type: "number",
            description: "The second number to add",
          },
        },
        required: ["x", "y"],
      },
    },
  },
]

const tools_map = (): { [key: string]: (args: any) => any } => ({
  add: add,
})

export async function init_db({
  filename = "chains.db",
}: {
  filename: string
}): Promise<void> {
  const db = await open({
    filename: filename,
    driver: sqlite3.Database,
  })

  await db.exec(`DROP TABLE IF EXISTS function_calls`)
  await db.exec(`CREATE TABLE IF NOT EXISTS function_calls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chain_id INTEGER NOT NULL,
    external_function_call_id TEXT NOT NULL,
    function_name TEXT NOT NULL,
    function_args JSON NOT NULL,
    result JSON,
    added_to_chain_at DATETIME,
    child_chain_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chain_id) REFERENCES chains (id),
    FOREIGN KEY (child_chain_id) REFERENCES chains (id),
    UNIQUE (chain_id, external_function_call_id)
  )`)

  await db.exec(`DROP TABLE IF EXISTS agents`)
  await db.exec(`CREATE TABLE IF NOT EXISTS agents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    system_prompt TEXT NOT NULL,
    tools JSON NOT NULL,
    delegation_tool_name TEXT NOT NULL,
    delegation_tool_description TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  )`)

  await db.exec(`DROP TABLE IF EXISTS chains`)
  await db.exec(`CREATE TABLE IF NOT EXISTS chains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    messages JSON NOT NULL,
    agent_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'awaiting_llm_processing',
    parent_function_call_id INTEGER,
    FOREIGN KEY (parent_function_call_id) REFERENCES function_calls (id)
  )`)
}

export async function chainWorker(filename: string = "chains.db"): Promise<void> {
  const openai = new OpenAI()
  await init_db({ filename })
  const store = await SQLiteStore({ filename })
  const db = await open({
    filename: filename,
    driver: sqlite3.Database,
  })
  const service = new ChainService({
    store, db, openai, toolsMap: {
      "add": add,
    }
  })

  const calculatorAgent: Agent = {
    name: "calculator_operator",
    system_prompt: "You are a skilled calculator operator",
    tools: add_tools(),
    delegation_tool: {
      description:
        "a skilled calculator operator that can perform various arithmetic operations",
    },
  }
  const managerAgent = {
    name: "project_manager",
    system_prompt:
      `You are a project manager and assistant that can delegate tasks to other agents to accomplish a goal.

      always keep delegated tasks as small as possible
      
      `,
    tools: [
      agent_to_tool(calculatorAgent),
    ],
    delegation_tool: {
      description:
        "a project manager that can delegate tasks to other agents to accomplish a goal",
    },
  }


  await store.insertAgent(calculatorAgent)
  await store.insertAgent(managerAgent)

  const addChain = await service.storeChain({
    userMessage: "Add (2 + 5) and (3 + 4), then add the results",
    agent_name: "project_manager",
  })


  service.watchChains()

  // const answer = await service.awaitChainAnswer({ id: addChain.id })
  // console.log(`ANSWER: ${answer}`)

  // const hnPosts = await fetchHnPostsLastDay()
  // await service.storeChain({
  //   userMessage: `
  //   research this hackernews post to find a contact email: ${JSON.stringify(hnPosts[0])}

  //   when you have researched the post, send a friendly email to the user along the lines of:

  //   subject: saw $PROJECT, looks cool

  //   body: yo - $PROJECT looks cool, love the idea of $FEATURE - can we chat?

  //   - dex @ humanlayer.dev
  //   `,
  //   agent_name: "project_manager"
  // })

}

export class ChainService {
  public store: Store
  public db: any;
  public openai: OpenAI;
  public toolsMap: Record<string, (kwargs: any) => any>;

  constructor({
    store,
    db,
    openai,
    toolsMap = tools_map()
  }: {
    store: Store;
    db: any;
    openai: OpenAI;
    toolsMap?: Record<string, (kwargs: any) => any>
  }) {
    this.store = store;
    this.db = db;
    this.openai = openai;
    this.toolsMap = toolsMap;
  }

  async storeChain({
    userMessage,
    agent_name = "project_manager",
    parent_function_call_id,
  }: {
    userMessage: string
    agent_name: string
    parent_function_call_id?: number
  }): Promise<Chain> {
    const agent = await this.store.getAgent(agent_name)

    const messages: any[] = [
      {
        role: "system",
        content: agent.system_prompt,
      },
      {
        role: "user",
        content: userMessage,
      },
    ]

    return await this.store.insertChain(messages, agent.id, parent_function_call_id)
  }

  async awaitChain({ id, poll_interval = 10000 }: { id: number, poll_interval?: number }) {
    let chain = await this.store.getChain(id)
    while (chain.status !== "stop_awaiting_user") {
      await new Promise(resolve => setTimeout(resolve, poll_interval))
      chain = await this.store.getChain(id)
    }
    return chain
  }
  async awaitChainAnswer({ id, poll_interval = 10000 }: { id: number, poll_interval?: number }): Promise<string> {
    const chain = await this.awaitChain({ id, poll_interval });
    const messages = JSON.parse(chain.messages);
    const lastMessage = messages[messages.length - 1];
    return lastMessage.content;
  }

  async executeWaitingFunctionCalls() {
    const waitingFunctionCalls = await this.store.getWaitingFunctionCalls()
    for (const function_call of waitingFunctionCalls) {
      const function_name = function_call.function_name
      const function_args = JSON.parse(function_call.function_args)
      if (this.toolsMap[function_name]) {
        const function_result = JSON.stringify(
          await this.toolsMap[function_name](function_args)
        )
        await this.store.updateFunctionCallResult(function_call.id, function_result)
        logger.info(
          `tool:${function_call.external_function_call_id} <-- ${function_call.function_name}(${function_call.function_args}) = ${function_result}`
        )
        continue
      }
      if (function_name.startsWith("delegate_to_")) {
        const agent_name = function_name.split("delegate_to_")[1]
        const agent = await this.db.get("SELECT * FROM agents WHERE name = ?", [
          agent_name,
        ])
        if (!agent) {
          logger.error(
            `agent ${agent_name} not found for ${function_name}(${function_args})`
          )
          await this.db.run(
            "UPDATE function_calls SET result = ? WHERE id = ?",
            [
              JSON.stringify({ error: `agent ${agent_name} not found` }),
              function_call.id,
            ]
          )
          continue
        }

        // insert a new chain with this function call as the parent
        const child_chain = await this.storeChain({
          userMessage: function_args.message,
          agent_name,
          parent_function_call_id: function_call.id,
        })

        await this.db.run(`UPDATE function_calls 
          SET child_chain_id = ? 
          WHERE id = ?`, [
          child_chain.id,
          function_call.id,
        ])
      } else {
        logger.error(
          `function ${function_name} not found for ${function_call.external_function_call_id}`
        )
        await this.db.run("UPDATE function_calls SET result = ? WHERE id = ?", [
          JSON.stringify({ error: `function ${function_name} not found` }),
          function_call.id,
        ])
      }
    }
  }
  async checkWaitingchains() {
    const waitingchains = await this.store.getChainsAwaitingLLMProcessing()

    for (const chain of waitingchains) {
      const parsedChain = JSON.parse(chain.messages)
      logger.info(`LLM <-- chain:${chain.id} ${chain.messages}`)
      // Set the status to llm_processing
      await this.store.updateChainStatus(chain.id, "llm_processing")
      const completion = await this.openai.chat.completions.create({
        model: "gpt-4o-mini",
        messages: parsedChain,
        tools: JSON.parse(chain.tools),
        tool_choice: "auto",
      })

      parsedChain.push(completion.choices[0].message)

      if (completion.choices[0].message.tool_calls) {
        await this.db.run(
          "UPDATE chains SET status = ?, messages = ? WHERE id = ?",
          ["awaiting_function_call", JSON.stringify(parsedChain), chain.id]
        )
      } else {
        await this.db.run(
          "UPDATE chains SET status = ?, messages = ? WHERE id = ?",
          ["stop_awaiting_user", JSON.stringify(parsedChain), chain.id]
        )
      }
      logger.info(
        `LLM --> chain:${chain.id} ${JSON.stringify(parsedChain.slice(-1)[0])}`
      )
    }
    const chainsAwaitingFunctionCall =
      await this.store.getChainsAwaitingFunctionCall()

    for (const chain of chainsAwaitingFunctionCall) {
      const parsedChain = JSON.parse(chain.messages)
      const function_calls = parsedChain.slice(-1)[0].tool_calls
      if (!function_calls) {
        logger.error(
          `chain:${chain.id} --> tool:UNKNOWN had no function calls`
        )
        continue
      }
      for (const function_call of function_calls) {
        // Insert function call into the function_calls table
        const result = await this.db.run(
          `INSERT INTO function_calls
           (chain_id, external_function_call_id, function_name, function_args)
           VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING`,
          [
            chain.id,
            function_call.id,
            function_call.function.name,
            function_call.function.arguments,
          ]
        )
        if (result.changes !== 0) {
          logger.info(
            `chain:${chain.id} --> tool:${function_call.id}, ${function_call.function.name}(${function_call.function.arguments})`
          )
        }
      }
    }
  }

  async propagateFunctionCallResultsToThread() {
    const function_calls = await this.db.all(
      `
        SELECT fc.*, c.messages, a.name as agent_name
        FROM function_calls fc
               JOIN chains c ON fc.chain_id = c.id
               JOIN agents a ON c.agent_id = a.id
        WHERE c.status = 'awaiting_function_call'
          AND fc.added_to_chain_at IS NULL
          AND NOT EXISTS (SELECT 1
                          FROM function_calls fc2
                          WHERE fc2.chain_id = c.id
                            AND fc2.result IS NULL)
      `
    )

    // Group function calls by chain_id
    const groupedFunctionCalls: { [key: string]: any[] } =
      function_calls.reduce((acc: { [key: string]: any[] }, call: any) => {
        if (!acc[call.chain_id]) {
          acc[call.chain_id] = []
        }
        acc[call.chain_id].push(call)
        return acc
      }, {})

    // Process each chain
    for (const [chain_id, calls] of Object.entries(groupedFunctionCalls)) {
      const parsedChain = JSON.parse(calls[0].messages)

      // Add function call results to the chain
      for (const call of calls) {
        parsedChain.push({
          role: "tool",
          name: call.function_name,
          content: JSON.stringify(call.result),
          tool_call_id: call.external_function_call_id,
        })
        logger.info(
          `chain:${chain_id}::agent:${call.agent_name} <-- ${call.function_name}(${call.function_args}) = ${call.result}`
        )
      }

      // Update the chain to be ready for processing...probably need a txn around this function
      await this.db.run("BEGIN TRANSACTION")
      try {
        await this.db.run(
          `UPDATE function_calls
           SET added_to_chain_at = CURRENT_TIMESTAMP
           WHERE id IN (${Array(calls.length).fill("?").join(",")})`,
          calls.map((call) => call.id)
        )
        await this.db.run(
          "UPDATE chains SET messages = ?, status = ? WHERE id = ?",
          [JSON.stringify(parsedChain), "awaiting_llm_processing", chain_id]
        )
        await this.db.run("COMMIT")
      } catch (error) {
        await this.db.run("ROLLBACK")
        throw error
      }
    }
  }
  async watchChildChainsForFunctionCallResults() {
    const childChains = await this.db.all(`
      SELECT c.messages,
             fc.id  as function_call_id,
             fc.external_function_call_id,
             fc.function_name,
             fc.function_args,
             fc.result,
             a.name as agent_name
      FROM chains c
             JOIN function_calls fc ON c.parent_function_call_id = fc.id
             JOIN agents a ON c.agent_id = a.id
      WHERE c.parent_function_call_id IS NOT NULL
        AND fc.result IS NULL
        AND c.status = 'stop_awaiting_user'
    `)
    for (const chain of childChains) {
      const parsedChain = JSON.parse(chain.messages)
      const lastMessage = parsedChain.slice(-1)[0]
      if (lastMessage.role === "assistant" && !lastMessage.tool_calls) {
        logger.info(
          `tool:${chain.external_function_call_id} <-- agent:${chain.agent_name} : ${lastMessage.content}`
        )
        await this.db.run("UPDATE function_calls SET result = ? WHERE id = ?", [
          lastMessage.content,
          chain.function_call_id,
        ])
      }
    }
  }

  watchChains() {
    setInterval(async () => {
      await this.checkWaitingchains()
      await this.executeWaitingFunctionCalls()
      await this.propagateFunctionCallResultsToThread()
      await this.watchChildChainsForFunctionCallResults()
    }, 1000)
  }
}

export interface Store {
  updateChainStatus(id: number, status: string): Promise<void>
  insertChain(chain: any[], agentId: number, parentFunctionCallId?: number): Promise<Chain>
  getAgent(name: string): Promise<Agent & { id: number }>
  insertAgent(agent: Agent): Promise<void>
  listChains(): Promise<(Chain & { parent_external_function_call_id: string })[]>
  getChainsAwaitingLLMProcessing(): Promise<(Chain & { tools: string })[]>
  getChainsAwaitingFunctionCall(): Promise<(Chain & { tools: string })[]>
  getWaitingFunctionCalls(): Promise<FunctionCall[]>
  getChain(id: number): Promise<Chain>
  updateFunctionCallResult(id: number, result: string): Promise<void>
}

export const SQLiteStore = async ({
  filename = "chains.db",
}: {
  filename: string
}): Promise<Store> => {
  const db = await open({
    filename,
    driver: sqlite3.Database,
  })

  return {
    async updateChainStatus(id: number, status: string) {
      await db.run("UPDATE chains SET status = ? WHERE id = ?", [status, id])
    },
    async insertChain(
      chain: any[],
      agentId: number,
      parentFunctionCallId?: number
    ): Promise<Chain> {
      const result = await db.run(
        `INSERT INTO chains (messages, agent_id, parent_function_call_id) 
        VALUES (?, ?, ?)`,
        [JSON.stringify(chain), agentId, parentFunctionCallId || null]
      )

      if (!result.lastID) {
        throw new Error("Failed to insert chain")
      }
      return {
        id: result.lastID,
        messages: JSON.stringify(chain),
        created_at: new Date().toISOString(),
        status: "awaiting_llm_processing",
        parent_function_call_id: parentFunctionCallId || null,
      }
    },

    async getAgent(name: string): Promise<Agent & { id: number }> {
      const agent = await db.get("select * from agents where name = ?", [name])
      if (!agent) {
        throw new Error(`Agent not found: ${name}`)
      }
      return {
        id: agent.id,
        name: agent.name,
        system_prompt: agent.system_prompt,
        tools: JSON.parse(agent.tools),
        delegation_tool: {
          description: agent.delegation_tool_description,
        },
      }
    },

    async insertAgent(agent: Agent) {
      await db.run(
        `INSERT INTO agents (name, system_prompt, tools, delegation_tool_name, delegation_tool_description) VALUES (?, ?, ?, ?, ?)`,
        [
          agent.name,
          agent.system_prompt,
          JSON.stringify(agent.tools),
          `delegate_to_${agent.name}`,
          agent.delegation_tool?.description,
        ]
      )
    },
    async updateFunctionCallResult(id: number, result: string): Promise<void> {
      await db.run("UPDATE function_calls SET result = ? WHERE id = ?", [
        result,
        id,
      ])
    },


    async listChains(): Promise<(Chain & { parent_external_function_call_id: string })[]> {
      const chains = await db.all(
        `SELECT c.*, a.name as agent_name,
          fc.external_function_call_id as parent_external_function_call_id
        FROM chains c
          LEFT JOIN function_calls fc ON c.parent_function_call_id = fc.id
          LEFT JOIN agents a ON c.agent_id = a.id
        ORDER BY created_at ASC`
      )
      return chains.map(({ agent_name, id, messages, created_at, parent_function_call_id, status, parent_external_function_call_id }: Chain & {
        parent_external_function_call_id: string,
        agent_name: string,
      }) => ({
        id,
        messages: JSON.parse(messages),
        agent_name,
        created_at,
        parent_function_call_id,
        status,
        parent_external_function_call_id,
      }))
    },
    async getChainsAwaitingLLMProcessing(): Promise<
      (Chain & { tools: string })[]
    > {
      return await db.all(
        `
        SELECT c.*, a.tools
        FROM chains c
               JOIN agents a ON c.agent_id = a.id
        WHERE c.status IN (?, ?)
        ORDER BY c.created_at ASC
      `,
        ["awaiting_llm_processing"]
      )
    },
    async getChainsAwaitingFunctionCall(): Promise<
      (Chain & { tools: string })[]
    > {
      return await db.all(
        `
        SELECT c.*, a.tools
        FROM chains c
               JOIN agents a ON c.agent_id = a.id
        WHERE c.status IN (?, ?)
        ORDER BY c.created_at ASC
      `,
        ["awaiting_llm_processing", "awaiting_function_call"]
      )
    },
    async getWaitingFunctionCalls(): Promise<FunctionCall[]> {
      return await db.all(
        `SELECT * FROM function_calls 
        WHERE 
            result IS NULL 
            AND child_chain_id IS NULL
        ORDER BY created_at ASC`
      )
    },
    async getChain(id: number): Promise<Chain> {
      const chain = await db.get("SELECT * FROM chains WHERE id = ?", [id])
      if (!chain) {
        throw new Error(`Chain not found: ${id}`)
      }
      return {
        ...chain,
        chain: JSON.parse(chain.messages),
      }
    },
  }
}

type FunctionCall = {
  id: number
  chain_id: number
  external_function_call_id: string
  function_name: string
  function_args: string
}
