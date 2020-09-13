import React from 'react';

import 'xterm/css/xterm.css';
import { Terminal } from 'xterm';
import { spawnTerminal, mkdirAll } from './GoWASM';

export default function({ args, ...props }) {
  const [term] = React.useState(() => {
    const term = new Terminal()
    const cwd = "/home/me/playground"
    mkdirAll(cwd).then(() =>
      spawnTerminal(term, { args, cwd }))
    return term
  })
  const elem = React.useRef(null)
  React.useEffect(() => {
    if (elem) {
      term.open(elem.current)
      term.setOption('cursorBlink', true)
      term.focus()
    }
  }, [elem, term])

  return <div ref={elem} {...props} />
}
