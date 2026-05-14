import { useEffect, useState } from 'react';
import { API } from './api';
import './LatencyDisplay.css';

function SVGBars({ numberOfBars } : { numberOfBars: number }) {
  const colourArray = ["red", "orange", "green", "green"]
  const colour = colourArray[Math.max(Math.min(numberOfBars, colourArray.length), 1) - 1]
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


function LatencyBars({ ping }: { ping: number }) {
  const pingThresholds = [50, 250, 500]
  let pingLevel = 0

  for (; pingLevel < pingThresholds.length; pingLevel++) {
    if (ping <= pingThresholds[pingLevel]) {
      break
    }
  }

  return (
    <SVGBars numberOfBars={4 - pingLevel}/>
  )
}

export function LatencyDisplay() {
  const [ping, setPing] = useState<number | null>(null)

  useEffect(() => {
    let cancelled = false

    async function measurePing() {
      try {
        const start = performance.now()
        await fetch(API.validateSession, { method: "POST", credentials: "include" })
        const elapsed = Math.round(performance.now() - start)
        if (!cancelled) setPing(elapsed)
      } catch {
        if (!cancelled) setPing(null)
      }
    }

    measurePing()
    const interval = setInterval(measurePing, 30_000)

    return () => {
      cancelled = true
      clearInterval(interval)
    }
  }, [])

  return (
    <div className='latencyContainer'>
      <div className='latencyText'>
        <div>
          Ping: {ping == null ? "?" : ping + "ms"}
        </div>
      </div>

      <div className='latencyBars'>
        <LatencyBars ping={ping ?? 0}/>
      </div>
    </div>
  )
}
