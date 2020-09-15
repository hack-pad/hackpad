import React from 'react';

import 'xterm/css/xterm.css';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';

export default function({ args, ...props }) {
  const elem = React.useRef(null)
  React.useEffect(() => {
    if (elem) {
      newTerminal(elem)
    }
  }, [elem])

  return <div ref={elem} {...props} />
}

export function newTerminal(elem) {
  const fitAddon = new FitAddon()
  const term = new Terminal()
  term.loadAddon(fitAddon)

  term.open(elem)
  term.setOption('cursorBlink', true)
  term.focus()
  fitAddon.fit()
  if (window.ResizeObserver) {
    const observer = new ResizeObserver(() => fitAddon.fit())
    observer.observe(elem)
  } else {
    window.addEventListener('resize', () => fitAddon.fit())
  }
  return term
}
