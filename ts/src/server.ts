import express from "express"
import proxy from "express-http-proxy"

import { ChainService } from "./chain"
import { SQLiteStore } from "./chain"
import { open } from "sqlite"
import sqlite3 from "sqlite3"
import cors from "cors"

import { OpenAI } from "openai"

const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
})


const STATIC_ASSETS_PATH = process.env.STATIC_ASSETS_PATH
const PROXY_UI_DEV_SERVER = process.env.PROXY_UI_DEV_SERVER || "http://localhost:5173"

const app = express()
const port = process.env.UI_PORT || 4002

export const startServer = async () => {
  const filename = "chains.db"
  const store = await SQLiteStore({ filename })
  const db = await open({
    filename,
    driver: sqlite3.Database,
  })
  const service = new ChainService({ store, db, openai })

  app.use("/api", express.json())

  app.post(
    "/api/chains",
    async (req: express.Request, res: express.Response) => {
      try {
        const { userMessage, agent_name } = req.body
        await service.storeChain({ userMessage, agent_name })
        res.status(201).json({ message: "Chain stored successfully" })
      } catch (error) {
        console.error("Error storing chain:", error)
        res.status(500).json({ error: "Internal server error" })
      }
    }
  )

  app.get(
    "/api/chains",
    async (req: express.Request, res: express.Response) => {
      try {
        const chains = await store.listChains()
        res.json(chains)
      } catch (error) {
        console.error("Error listing chains:", error)
        res.status(500).json({ error: "Internal server error" })
      }
    }
  )

  app.get(
    "/api/chains/tree",
    async (req: express.Request, res: express.Response) => {
      try {
        const tree = null
        res.json(tree)
      } catch (error) {
        console.error("Error getting tree:", error)
        res.status(500).json({ error: "Internal server error" })
      }
    }
  )

  if (STATIC_ASSETS_PATH) {
    app.use("/", express.static(STATIC_ASSETS_PATH))
  } else {
    app.use("/", proxy(PROXY_UI_DEV_SERVER))
  }


  app.listen(port, () => {
    console.log(`Server running at http://localhost:${port}`)
  })
}
