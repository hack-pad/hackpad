import React from 'react';

import 'xterm/css/xterm.css';
import { Terminal } from 'xterm';
import { spawnTerminal } from './GoWASM';

export default function({ args }) {
  const [term] = React.useState(() => {
    const term = new Terminal()
    spawnTerminal(term, args)
    return term
  })
  const elem = React.useRef(null)
  React.useEffect(() => {
    if (elem) {
      term.open(elem.current)
      term.focus()
    }
  }, [elem, term])

  return <div ref={elem} />
}