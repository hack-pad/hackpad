import React from 'react';
import './App.css';

import { spawn, install } from './GoWASM';

function App() {
  React.useEffect(() => {
    install('editor').then(() => spawn('editor', 'editor'))
    //install('sh').then(() => spawn('sh', 'sh'))
  }, [])

  return (
    <div id="app">
      <div id="editor"></div>
      <div id="sh"></div>
    </div>
  );
}

export default App;
