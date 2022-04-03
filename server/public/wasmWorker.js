"use strict";

async function runWasm(params) {
  console.log("Loading Wasm:", params)
  self.importScripts("wasm/wasm_exec.js")
  const go = new Go()
  const result = await WebAssembly.instantiateStreaming(fetch(params.wasm), go.importObject)
  await go.run(result.instance)
  close()
}

const params = new URLSearchParams(self.location.search)
const paramsObj = {}
for (const [key, value] of params) {
  paramsObj[key] = value
}
runWasm(paramsObj)
