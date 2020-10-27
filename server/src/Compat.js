import React from 'react';
import './Compat.css';


export default function Compat() {
  const knownWorkingBrowsers = [
    'Chrome',
    'Firefox',
  ]
  if (knownWorkingBrowsers.includes(getBrowser())) {
    return null
  }
  return (
    <div className="compat">
      <p>Go Wasm may not work reliably in your browser.</p>
      <p>If you're seeing issues, try a recent version of {joinOr(knownWorkingBrowsers)}.</p>
    </div>
  )
}

function getBrowser() {
  if (window.navigator.vendor.match(/google/i)) {
    return 'Chrome'
  } else if (navigator.vendor.match(/apple/i)) {
    return 'Safari'
  } else if (navigator.userAgent.match(/firefox\//i)) {
    return 'Firefox'
  }
  return 'other';
}

function joinOr(arr) {
  if (arr.length === 1) {
    return arr[0]
  }
  if (arr.length === 2) {
    return `${arr[0]} or ${arr[1]}`
  }

  const commaDelimited = arr.slice(0, arr.length - 1).join(", ")
  return `${commaDelimited}, or ${arr[arr.length - 1]}`
}
