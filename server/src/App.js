import React from 'react';
import './App.css';

import './Tabs.css';
import "@fontsource/roboto";
import '@fortawesome/fontawesome-free/css/all.css';
import Compat from './Compat';
import Loading from './Loading';
import { install, run, observeGoDownloadProgress } from './Hackpad';
import { newEditor } from './Editor';
import { newTerminal } from './Terminal';

function App() {
  const [percentage, setPercentage] = React.useState(0);
  const [loading, setLoading] = React.useState(true);
  React.useEffect(() => {
    observeGoDownloadProgress(setPercentage)

    window.editor = {
      newTerminal,
      newEditor,
    }
    Promise.all([ install('editor'), install('sh') ])
      .then(() => {
        run('editor', '--editor=editor')
        setLoading(false)
      })
  }, [setLoading, setPercentage])

  return (
    <>
      { loading ? <>
        <Compat />
        <Loading percentage={percentage} />
      </> : null }
      <div id="app">
        <div id="editor"></div>
      </div>
    </>
  );
}

export default App;
