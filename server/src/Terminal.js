import React from 'react';

import 'xterm/css/xterm.css';
import { Terminal as XTerminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { listenColorScheme } from './ColorScheme';

export default function Terminal({ args, ...props }) {
  const elem = React.useRef(null)
  React.useEffect(() => {
    if (elem) {
      newTerminal(elem)
    }
  }, [elem])

  return <div ref={elem} {...props} />
}

const fontScale = 0.85

export function newTerminal(elem) {
  const fitAddon = new FitAddon()
  const term = new XTerminal({
  })
  term.loadAddon(fitAddon)

  const dark = "rgb(33, 33, 33)"
  const light = "white"
  listenColorScheme({
    light: () => term.setOption('theme', {
      background: light,
      foreground: dark,
      cursor: dark,
    }),
    dark: () => term.setOption('theme', {
      background: dark,
      foreground: light,
      cursor: light,
    }),
  })

  term.open(elem)
  term.setOption('cursorBlink', true)
  term.focus()
  const fit = () => {
    const fontSize = parseFloat(getComputedStyle(elem).fontSize)
    term.setOption('fontSize', fontSize * fontScale)
    fitAddon.fit()
  }

  fit()
  if (window.ResizeObserver) {
    const parent = elem.parentNode
    const observer = new ResizeObserver(() => {
      if (! elem.parentNode) {
        // elem removed from DOM
        observer.unobserve(parent)
        return
      }
      if (elem.classList.contains("active")) {
        fit()
      }
    })
    observer.observe(parent)
  } else {
    window.addEventListener('resize', fit)
  }
  return term
}
