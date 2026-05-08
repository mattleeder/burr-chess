import { useContext, useEffect, useState } from "react";
import { GameContext, GameWrapper } from "./GameContext";
import { ChessBoard } from "./ChessBoard";
import { API } from "../api";


function sleep(delayMs: number){
  return new Promise((resolve) => setTimeout(resolve, delayMs));
}

async function fetchHighestEloMatchID(signal: AbortSignal) {
  let retryDelayMs = 1000
  const retryDelayMaxMs = 30_000 // 30s

  while (true) {

    try {
      const response = await fetch(API.fetchHighestElo, {
        signal,
        "method": "GET",
        "mode": "cors",
      })

      if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
      }

      // 204 is response.ok == true, as there is no content we should retry
      if (response.status != 204) {
        const data = await response.json()
        return data["matchID"]
      }      
    }
      
    catch (error: unknown) {
      if (error instanceof Error) {
        if (error.name == "AbortError") {
          return
        }
        console.error(error.message)
      } else {
        console.error(error)
      }
    }

    await sleep(retryDelayMs)
    retryDelayMs = Math.min(retryDelayMs * 2, retryDelayMaxMs)
  }
}

function GameOverListener({ callbackFunction }: { callbackFunction: () => void }) {
  const gameContext = useContext(GameContext)
  const isGameOver = gameContext?.matchData.gameOverStatus !== undefined && gameContext?.matchData.gameOverStatus != 0

  useEffect(() => {
    if (isGameOver) {
      callbackFunction()
    }
  }, [isGameOver, callbackFunction])

  return (
    <></>
  )
}

export function HomepageMatch() {
  const [matchID, setMatchID] = useState<undefined | string>(undefined)
  const parsedTimeFormatInMilliseconds = 0

  const onMatchEnd = () => {
    setMatchID(undefined)
  }

  useEffect(() => {
    const controller = new AbortController()
    const signal = controller.signal
    if (matchID === undefined) {
      fetchHighestEloMatchID(signal).then((data) => setMatchID(data))
    }
    return () => {
      controller.abort()
    }
  }, [matchID])

  if (matchID === undefined) {
    return (
      <div className='chessboard' />
    )
  }

  return (
    <GameWrapper matchID={matchID as string} timeFormatInMilliseconds={parsedTimeFormatInMilliseconds}>
      <div className='chessMatch'>
        <GameOverListener callbackFunction={onMatchEnd} />
        <ChessBoard resizeable={false} defaultWidth={300} chessboardContainerStyles={{transform: "translate(-800px, 200px)"}} enableClicking={false}/>
      </div>
    </GameWrapper>
  )
}