import { chainWorker } from "./chain"
import { startServer } from "./server"

startServer()

chainWorker().catch((e: any) => {
  console.trace()
  console.error(e)
})


