import React from 'react';
import './App.css';

import { install, run } from './GoWASM';
import { newTerminal } from './Terminal';
import { newEditor } from './Editor';

function App() {
  React.useEffect(() => {
    window.editor = {
      newTerminal,
      newEditor,
    }
    Promise.all([ install('editor'), install('sh') ])
      .then(() => {
        run('editor', '--editor=editor')
      })
  }, [])

  return (
    <div id="app">
      <div id="editor"></div>
    </div>
  );
}

export default App;
