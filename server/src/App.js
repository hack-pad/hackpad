import React from 'react';
import './App.css';

import './Tabs.css';
import '@fortawesome/fontawesome-free/css/all.css';
import Compat from './Compat';
import Loading from './Loading';
import { install, run } from './GoWASM';
import { newEditor } from './Editor';
import { newTerminal } from './Terminal';

function App() {
  const [loading, setLoading] = React.useState(true);
  React.useEffect(() => {
    window.editor = {
      newTerminal,
      newEditor,
    }
    Promise.all([ install('editor'), install('sh') ])
      .then(() => {
        run('editor', '--editor=editor')
        setLoading(false)
      })
  }, [setLoading])

  return (
    <>
      {loading ? <Loading /> : null}
      <div id="app">
        <Compat />
        <div id="editor"></div>
      </div>
    </>
  );
}

export default App;
