import React from 'react';
import './App.css';

import { install, run } from './GoWASM';
import Terminal from './Terminal';

function App() {
  const [createTerm, setCreateTerm] = React.useState(false)
  React.useEffect(() => {
    Promise.all([ install('editor'), install('sh') ])
      .then(() => {
        run('editor', '--editor=editor', '--console=build-console', '--console-tab=editor-tab-build')
        setCreateTerm(true)
      })
  }, [])

  const tabs = [
    'Terminal',
    'Build',
  ]
  const [activeTab, setActiveTab] = React.useState(0)

  return (
    <div id="app">
      <div id="editor"></div>
      <div className="consoles tabs">
        <nav>
          <ul>
            {tabs.map((tab, i) => {
              let className = "tab-title"
              if (i === activeTab) {
                className += " active"
              }
              return (
                <li key={i}>
                  <button id={"editor-tab-" + tab.toLowerCase()} className={className} onClick={() => setActiveTab(i)}>{tab}</button>
                </li>
              )
            })}
          </ul>
        </nav>
        <div className={activeTab === 0 ? "tab active" : "tab"}>
          {createTerm ?
              <Terminal args={['sh']} className="tab-contents" />
          : null }
        </div>
        <div className={activeTab === 1 ? "tab active" : "tab"}>
          <div id="build-console" className="tab-contents" />
        </div>
      </div>
    </div>
  );
}

export default App;
