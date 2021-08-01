import WebAssembly from './WebAssembly';
import 'whatwg-fetch';

const Go = window.Go; // loaded from wasm_exec.js script in index.html

let overlayProgress = 0;
let progressListeners = [];

async function init() {
  const startTime = new Date().getTime()
  const go = new Go();
  const cmd = await WebAssembly.instantiateStreaming(fetch(`wasm/main.wasm`), go.importObject)
  go.env = {
    'GOMODCACHE': '/home/me/.cache/go-mod',
    'GOPROXY': 'https://proxy.golang.org/',
    'GOROOT': '/usr/local/go',
    'HOME': '/home/me',
    'PATH': '/bin:/home/me/go/bin:/usr/local/go/bin/js_wasm:/usr/local/go/pkg/tool/js_wasm',
  }
  go.run(cmd.instance)
  const { hackpad, fs } = window
  console.debug(`hackpad status: ${hackpad.ready ? 'ready' : 'not ready'}`)

  const mkdir = promisify(fs.mkdir)
  await mkdir("/bin", {mode: 0o700})
  await hackpad.overlayIndexedDB('/bin', {cache: true})
  await hackpad.overlayIndexedDB('/home/me')
  await mkdir("/home/me/.cache", {recursive: true, mode: 0o700})
  await hackpad.overlayIndexedDB('/home/me/.cache', {cache: true})

  await mkdir("/usr/local/go", {recursive: true, mode: 0o700})
  await hackpad.overlayTarGzip('/usr/local/go', 'wasm/go.tar.gz', {
    persist: true,
    skipCacheDirs: [
      '/usr/local/go/bin/js_wasm',
      '/usr/local/go/pkg/tool/js_wasm',
    ],
    progress: percentage => {
      overlayProgress = percentage
      progressListeners.forEach(c => c(percentage))
    },
  })

  console.debug("Startup took", (new Date().getTime() - startTime) / 1000, "seconds")
}

const initOnce = init(); // always wait on this to ensure hackpad window object is ready

export async function install(name) {
  await initOnce
  return window.hackpad.install(name)
}

export async function run(name, ...args) {
  const process = await spawn({ name, args })
  return await wait(process.pid)
}

export async function wait(pid) {
  await initOnce
  const { child_process } = window
  return await new Promise((resolve, reject) => {
    child_process.wait(pid, (err, process) => {
      if (err) {
        reject(err)
        return
      }
      resolve(process)
    })
  })
}

export async function spawn({ name, args, ...options }) {
  await initOnce
  const { child_process } = window
  return await new Promise((resolve, reject) => {
    const subprocess = child_process.spawn(name, args, options)
    if (subprocess.error) {
      reject(new Error(`Failed to spawn command: ${name} ${args.join(" ")}: ${subprocess.error}`))
      return
    }
    resolve(subprocess)
  })
}

export async function spawnTerminal(term, options) {
  await initOnce
  const { hackpad } = window
  return hackpad.spawnTerminal(term, options)
}

export async function mkdirAll(path) {
  await initOnce
  const { fs } = window
  fs.mkdirSync(path, { recursive: true, mode: 0o755 })
}

export function observeGoDownloadProgress(callback) {
  progressListeners.push(callback)
  callback(overlayProgress)
}

function promisify(fn) {
  return (...args) => {
    return new Promise((resolve, reject) => {
      const newArgs = [...args]
      newArgs.push((err, ...results) => {
        if (err) {
          reject(err)
        } else {
          resolve(results)
        }
      })
      fn(...newArgs)
    })
  }
}
