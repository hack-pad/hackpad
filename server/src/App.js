import React from 'react';
import './App.css';

import { install } from './GoWASM';
import Terminal from './Terminal';

function App() {
  const [showTerm, setShowTerm] = React.useState(false)
  React.useEffect(() => {
    install('sh').then(() => setShowTerm(true))
  }, [])


  return (
    <div id="app">
      <div id="editor"></div>
      <div id="sh"></div>
      {showTerm ?
        <Terminal args={['/bin/sh']} />
      : null }
    </div>
  );
}

export default App;
