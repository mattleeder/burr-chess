import { useContext } from "react";
import { useParams, useLocation } from "react-router-dom"
import { GameWrapper, GameContext } from "./GameContext";
import { ChessBoard } from "./ChessBoard";
import { GameInfoTile } from "./GameInfoTile";

function MatchRoomContent() {
  const game = useContext(GameContext)
  if (!game) throw new Error("MatchRoomContent must be used within a GameWrapper")

  if (game.wsConnectionFailed) {
    return (
      <div className='chessMatch'>
        <p>Connection lost. Please refresh the page to rejoin the game.</p>
      </div>
    )
  }

  return (
    <div className='chessMatch'>
      <ChessBoard resizeable={true} defaultWidth={800} enableClicking={true}/>
      <GameInfoTile />
    </div>
  )
}

export function MatchRoom() {
  const { matchid } = useParams()
  const location = useLocation();
  const { timeFormatInMilliseconds } = location.state || {};
  const parsedTimeFormatInMilliseconds = parseInt(timeFormatInMilliseconds)

  return (
    <GameWrapper matchID={matchid as string} timeFormatInMilliseconds={parsedTimeFormatInMilliseconds}>
      <MatchRoomContent />
    </GameWrapper>
  )
}