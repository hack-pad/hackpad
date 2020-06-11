let hash = null;
const timeoutMillis = 2000
const poll = () => {
    fetch("/api/build").then(resp => resp.json()).then(body => {
		const newHash = body.BuildHash;
        if (hash == null) {
            hash = newHash;
        } else if (newHash != hash) {
			window.location.reload()
		}
	})
}
poll()
window.setInterval(poll, timeoutMillis)
