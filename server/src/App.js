import React from 'react';
import './App.css';

import './Tabs.css';
import Compat from './Compat';
import { install, run } from './GoWASM';
import { newEditor } from './Editor';
import { newTerminal } from './Terminal';

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
      <Compat />
      <div id="editor"></div>
    </div>
  );
}

export default App;
