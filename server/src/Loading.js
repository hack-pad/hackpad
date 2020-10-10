import React from 'react';
import './Loading.css';


export default function Loading() {
  return (
    <div className="app-loading">
      <div className="app-loading-center">
        <div className="app-loading-spinner">
          <span className="fa fa-spin fa-circle-notch" />
        </div>
        <p>
          installing <span className="app-title">
            <span className="app-title-go">go</span> <span className="app-title-wasm">wasm</span>
          </span>
        </p>
        <p><em>please wait...</em></p>
      </div>
    </div>
  )
}
