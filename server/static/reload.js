let hash = null;
const timeoutMillis = 2000
let changedHashRecently = false
const poll = () => {
    fetch("/api/build").then(resp => resp.json()).then(body => {
        const newHash = body.BuildHash;
        if (hash == null) {
            hash = newHash;
        } else if (newHash === hash && changedHashRecently) {
            window.location.reload()
        } else if (newHash !== hash) {
            changedHashRecently = true
            hash = newHash
        }
    })
}
poll()
window.setInterval(poll, timeoutMillis)
