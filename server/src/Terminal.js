import React from 'react';

import 'xterm/css/xterm.css';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { spawnTerminal, mkdirAll } from './GoWASM';

export default function({ args, ...props }) {
  const [fitAddon] = React.useState(new FitAddon())
  const [term] = React.useState(() => {
    const term = new Terminal()
    term.loadAddon(fitAddon)

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
      fitAddon.fit()
      window.addEventListener('resize', () => fitAddon.fit())
    }
  }, [elem, term, fitAddon])

  return <div ref={elem} {...props} />
}
