

export function listenColorScheme({ light, dark }) {
  const fn = mq => {
    const darkTheme = mq.matches
    if (darkTheme) {
      dark()
    } else {
      light()
    }
  }
  observeTheme(fn)
  fn(getMedia())
}

function getMedia() {
  return window.matchMedia("(prefers-color-scheme: dark)")
}

function init() {
  if (! window.matchMedia) {
    return () => false
  }
  return fn => getMedia().addListener(fn)
}

const observeTheme = init();
