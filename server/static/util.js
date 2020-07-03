async function run(name, ...args) {
    const downloadPath = `/bin/${name}`
    if (! await exists(downloadPath)) {
        if (! await exists("/bin")) {
            fs.mkdirSync("/bin", 0o700)
        }
        console.debug("Downloading WASM binary:", name)
        const wasmBinary = await fetch(`${name}.wasm`)
        const buf = await wasmBinary.arrayBuffer()
        writeBytes(downloadPath, new Uint8Array(buf))
        console.debug("Done downloading:", name)
        fs.chmodSync(downloadPath, 0o700)
    }
    return await spawn(name, ...args)
}

function ls(path) {
    if (!path) {
        path = ""
    }
    console.log(fs.readdirSync(path))
}

function rm(path) {
    if (!path) {
        path = ""
    }
    fs.unlinkSync(path)
}

function cat(path) {
    const fd = fs.openSync(path)
    if (fd) {
        let s = ""
        const len = 4096
        let n = len
        while (n === len) {
            const buf = new Uint8Array(len)
            n = fs.readSync(fd, buf, 0, buf.length, null)
            s += new TextDecoder("utf-8").decode(buf).substr(0, n)
        }
        console.log(s)
        fs.closeSync(fd)
    }
}

function cd(path) {
    if (!!path) {
        path = ""
    }
    process.chdir(path)
}

function exists(path) {
    return new Promise(resolve => {
        fs.stat(path, err => resolve(!err))
    })
}

function fstat(fd) {
    return new Promise((resolve, reject) => {
        fs.fstat(fd, (err, stats) => {
            if (err) {
                reject(err)
                return
            }
            resolve(stats)
        })
    })
}

function readOnceFD(fd) {
    return new Promise((resolve, reject) => {
        const buf = new Uint8Array(4096)
        fs.read(fd, buf, 0, buf.length, null, (err, n, buf) => {
            if (err) {
                reject(err)
                return
            }
            const data = new TextDecoder("utf-8").decode(buf).substr(0, n)
            resolve([n, data])
        })
    })
}

function readFD(fd) {
    return new Promise((resolve, reject) => {
        let s = ""
        let intervalID = setInterval(async () => {
            let [n, data] = await readOnceFD(fd)
            if (n > 0) {
                s += data
            }
            const stat = await fstat(fd)
            if (stat.size === 0) {
                clearInterval(intervalID)
                resolve(s)
            }
        }, 1)
    })
}

function writeFD(fd, contents) {
    const buf = new TextEncoder("utf-8").encode(contents)
    return new Promise((resolve, reject) => {
        fs.write(fd, buf, 0, buf.length, null, (err, n, buf) => {
            if (err) {
                reject(err)
                return
            }
            fs.close(fd, err => {
                if (err) {
                    reject(err)
                    return
                }
                resolve([n, buf])
            })
        })
    })
}

function writeBytes(path, contents) {
    const fd = fs.openSync(path, fs.constants.O_WRONLY | fs.constants.O_CREAT | fs.constants.O_TRUNC)
    if (fd) {
        fs.writeSync(fd, contents, 0, contents.length, null)
        fs.closeSync(fd)
    }
}

function write(path, contents) {
    const fd = fs.openSync(path, fs.constants.O_WRONLY | fs.constants.O_CREAT | fs.constants.O_TRUNC)
    if (fd) {
        const buf = new TextEncoder("utf-8").encode(contents)
        fs.writeSync(fd, buf, 0, buf.length, null)
        fs.closeSync(fd)
    }
}

function pipe() {
    return new Promise((resolve, reject) => {
        fs.pipe((err, fds) => {
            if (err) {
                reject(err)
                return
            }
            resolve({ r: fds[0], w: fds[1] })
            return
        })
    })
}

function spawn(name, ...args) {
    return new Promise((resolve, reject) => {
        const subprocess = child_process.spawn(name, args)
        if (subprocess.error) {
            reject(new Error(`Failed to spawn command: ${name} ${args.join(" ")}: ${subprocess.error}`))
            return
        }
        child_process.wait(subprocess.pid, (err, process) => {
            if (err) {
                reject(err)
                return
            }
            resolve(process)
        })
    })
}

function sh(str) {
    let args = str.split(" ")
    const name = args[0]
    args = args.slice(1)
    return spawn(name, ...args)
}
