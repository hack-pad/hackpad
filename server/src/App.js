import React from 'react';
import './App.css';

import './Tabs.css';
import "@fontsource/roboto";
import '@fortawesome/fontawesome-free/css/all.css';
import Compat from './Compat';
import Loading from './Loading';
import { observeGoDownloadProgress } from './Hackpad';
import { newEditor } from './Editor';
import { newTerminal } from './Terminal';

function App() {
  const [percentage, setPercentage] = React.useState(0);
  React.useEffect(() => {
    observeGoDownloadProgress(setPercentage)

    window.editor = {
      newTerminal,
      newEditor,
    }
  }, [setPercentage])

  return (
    <>
      <Compat />
      <div id="app">
        <div id="editor"></div>
      </div>
    </>
  );
}

export default App;
