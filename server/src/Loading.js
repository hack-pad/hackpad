import React from 'react';
import './Loading.css';


export default function Loading() {
  return (
    <div className="loading">
      <div className="loading-center">
        <div className="loading-spinner">
          <span className="fa fa-spin fa-circle-notch" />
        </div>
        <p><em>Loading, please wait...</em></p>
      </div>
    </div>
  )
}
