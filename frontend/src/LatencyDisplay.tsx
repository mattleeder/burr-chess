import React, { useEffect, useState } from 'react';
import './LatencyDisplay.css';

function fetchUserPing() {
  return 10
}

function fetchServerLatency() {
  return 20
}

function SVGBars({ numberOfBars } : { numberOfBars: number }) {

  const colourArray = ["red", "orange", "green", "green"]
  const colour = colourArray[Math.max(Math.min(numberOfBars, 1), colourArray.length) - 1]
  const heightOne = numberOfBars >= 1 ? "25" : "0"
  const heightTwo = numberOfBars >= 2 ? "50" : "0"
  const heightThree = numberOfBars >= 3 ? "75" : "0"
  const heightFour = numberOfBars >= 4 ? "100" : "0"

  return (
    <svg width="100%" height="100%" xmlns="http://www.w3.org/2000/svg" viewBox='0 0 100 100'>
      <rect width="24" height={heightOne} x="0" y="75" fill={colour} />
      <rect width="24" height={heightTwo} x="25" y="50" fill={colour} />
      <rect width="24" height={heightThree} x="50" y="25" fill={colour} />
      <rect width="24" height={heightFour} x="75" y="0" fill={colour} />
      Sorry, your browser does not support inline SVG.
    </svg>
  )
}


function LatencyBars({ userPing, serverLatency }: { userPing: number, serverLatency: number }) {
  const pingThresholds = [50, 250, 500]
  let pingLevel = 0

  for (null; pingLevel < pingThresholds.length; pingLevel++) {
    if (userPing + serverLatency <= pingThresholds[pingLevel]) {
      break
    }
  }

  return (
    <SVGBars numberOfBars={4 - pingLevel}/>
  )

}

export function LatencyDisplay() {
  const [userPing, setUserPing] = useState<number | null>(null)
  const [serverLatency, setServerLatency] = useState<number | null>(null)

  useEffect(() => {
    setUserPing(fetchUserPing())
    setServerLatency(fetchServerLatency())
  }, [])

  return (
    <div className='latencyContainer'>
      <div className='latencyText'>
        <div>
          Ping: {userPing == null ? "?" : userPing + "ms"}
        </div>
        <div>
          Server: {serverLatency == null ? "?" : serverLatency + "ms"}
        </div>
      </div>

      <div className='latencyBars'>
        <LatencyBars userPing={userPing == null ? 0 : userPing} serverLatency={serverLatency == null ? 0 : serverLatency}/>
      </div>
    </div>
  )
}