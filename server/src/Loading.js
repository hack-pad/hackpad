import React from 'react';
import './Loading.css';


export default function Loading({ percentage }) {
  return (
    <div className="app-loading">
      <div className="app-loading-center">
        <div className="app-loading-spinner">
          { percentage !== undefined ?
            <span className="app-loading-percentage">{Math.round(percentage)}%</span>
          : null }
          <span className="fa fa-spin fa-circle-notch" />
        </div>
        <p>
          installing <span className="app-title">
            <span className="app-title-hack">hack</span><span className="app-title-pad">pad</span>
          </span>
        </p>
        <p><em>please wait...</em></p>
      </div>
    </div>
  )
}
