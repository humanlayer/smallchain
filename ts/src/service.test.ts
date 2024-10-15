import {ChainService} from "./chain";

test('ChainService constructor', async () => {
  var store = {} as any
  var db = {} as any
  var openai = {} as any
  var service = new ChainService({ store, db, openai })

  expect(service.store).toBeDefined()
  expect(service.store).toBe(store)
  expect(service.db).toBe(db)
  expect(service.openai).toBe(openai)
})
