import React, { useContext, useRef, useEffect, useState } from "react";
import { CornerUpLeft, Handshake, Flag, Microscope, ChevronFirst, ChevronLeft, ChevronRight, ChevronLast, AlignJustify } from "lucide-react";
import { PieceColour, PieceVariant, parseGameStateFromFEN } from "./ChessLogic";
import { GameContext, OpponentEventType, gameContext, boardHistory, SQLNullString } from "./GameContext";
import { variantToString } from "./ChessBoard";
import { PlayerInfoTileContext, PlayerInfoTileContextInterface } from "../PlayerInfoTile";
import "./GameInfoTile.css";


function isClockPaused(game: gameContext, colour: PieceColour) {
  if (!game) {
    return true
  }
  return game.matchData.gameOverStatus != 0 || game.matchData.activeColour != colour|| game.matchData.stateHistory.length <= 2
}

function isBlackClockPaused(game: gameContext) {
  return isClockPaused(game, PieceColour.Black)
}
  
function isWhiteClockPaused(game: gameContext) {
  return isClockPaused(game, PieceColour.White)
}
  
function sendDrawEvent(websocket: React.RefObject<WebSocket | null>) {
  if (!websocket) {
    console.error("Websocket is null")
    return
  }
  
  websocket.current?.send(JSON.stringify({
    "messageType": "playerEvent",
    "body": {
      "eventType": OpponentEventType.Draw,
    }
  }))
}
  
function sendResignEvent(websocket: React.RefObject<WebSocket | null>) {
  if (!websocket) {
    console.error("Websocket is null")
    return
  }
  
  websocket.current?.send(JSON.stringify({
    "messageType": "playerEvent",
    "body": {
      "eventType": OpponentEventType.Resign,
    }
  }))
}

function acceptEvent(game: gameContext, eventType: OpponentEventType) {
  if (game.webSocket == null) {
    console.error("Websocket is null")
    return
  }
  
  game.webSocket.current?.send(JSON.stringify({
    "messageType": "playerEvent",
    "body": {
      "eventType": eventType,
    }
  }))

  game.setOpponentEventType(OpponentEventType.None)
  
}
  
function declineEvent(game: gameContext, isThreefold = false) {
  if (game.webSocket == null) {
    console.error("Websocket is null")
    return
  }

  if (isThreefold) {
    game.setThreefoldRepetition(false)
    return
  }
  
  game.webSocket.current?.send(JSON.stringify({
    "messageType": "playerEvent",
    "body": {
      "eventType": OpponentEventType.Decline,
    }
  }))

  game.setOpponentEventType(OpponentEventType.None)
  
    
}

export function PingStatus({ connected, style }: { connected: boolean, style?: React.CSSProperties }) {
  const fill = connected ? "#52f407" : "#bbb"
  return (
    <svg style={style} viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
      <circle cx="10" cy="10" r="10" fill={fill} />
      Sorry, your browser does not support inline SVG.
    </svg> 
  )
}

function PlayerInfo({ connected, username }: { connected: boolean, username: SQLNullString }) {
  const playerInfoTile = useContext<PlayerInfoTileContextInterface>(PlayerInfoTileContext)

  // useEffect(() => {
  //   playerInfoTile?.lightFusePlayerInfoTile(username.String, event)
  // }, [username])

  return (
    <div className='playerInfo'>
      <div className="playerPingStatus">
        <PingStatus connected={connected} />
      </div>
      <div className='playerName'
        onMouseEnter={(event) => {
          if (username.Valid) {
            playerInfoTile?.spawnPlayerInfoTile(username.String, event)
          }
        }}
        onMouseLeave={(event) => playerInfoTile?.lightFusePlayerInfoTile(username.String, event)}
      >{username.Valid ? username.String : "Anon"}</div>
    </div>
  )
}

function updateActiveState(stateHistoryIndex: number, game: gameContext) {
  if (!game) {
    throw new Error("updateActiveState must be called within a GameContext")
  }
  if (stateHistoryIndex == game.matchData.activeMove) {
    return
  }
  if (stateHistoryIndex < 0 || game.matchData.stateHistory.length - 1 < stateHistoryIndex) {
    return
  }
  const activeMoveNumber = stateHistoryIndex
  const matchData = {
    ...game.matchData
  }
  matchData.activeMove = activeMoveNumber
  matchData.activeState = {
    board: parseGameStateFromFEN(matchData.stateHistory[activeMoveNumber]["FEN"])["board"],
    lastMove: matchData.stateHistory[activeMoveNumber]["lastMove"],
    FEN: matchData.stateHistory[activeMoveNumber]["FEN"],
    whitePlayerTimeRemainingMilliseconds: matchData.activeState.whitePlayerTimeRemainingMilliseconds,
    blackPlayerTimeRemainingMilliseconds: matchData.activeState.blackPlayerTimeRemainingMilliseconds,
  }
  game.setMatchData(matchData)
}

function formatDuration(durationInMilliseconds: number): string {
  if (durationInMilliseconds <= 0) {
    return "00:00:00.0"
  }
  const time = new Date(durationInMilliseconds);
  // const hours = time.getUTCHours();
  const minutes = time.getUTCMinutes();
  const seconds = time.getUTCSeconds();
  const milliseconds = time.getUTCMilliseconds();
    
  let result = String(minutes).padStart(2, "0") + ":" + String(seconds).padStart(2, "0")
  if (durationInMilliseconds < 10_000) {
    result += "." + milliseconds.toPrecision(1)[0]
  }
  return result;
}

function GameControls() {
  const game = useContext(GameContext)
  if (!game) {
    throw new Error("GameControls must be used within a GameContext")
  }
  
  return (
    <div className='gameControlsContainer'>
      <div className='spacer' />
      <div className='gameControlsButton'>
        <CornerUpLeft color='#000000' />
      </div>
      <div className='gameControlsButton'>
        <Handshake onClick={() => sendDrawEvent(game.webSocket)} color='#000000' />
      </div>
      <div className='gameControlsButton'>
        <Flag onClick={() => sendResignEvent(game.webSocket)} color='#000000' />
      </div>
      <div className='spacer' />
    </div>
  )
}
  
function Moves() {
  const game = useContext(GameContext)
  if (!game) {
    throw new Error('Move History must be used within a GameContext Provider');
  }
  
  const boardHistory = game.matchData.stateHistory
  const tableRef = useRef<HTMLDivElement | null>(null)
  const activeRef = useRef<HTMLTableCellElement | null>(null)
  
  
  useEffect(() => {
    if (activeRef.current) {
      activeRef.current.scrollIntoView({ behavior: "auto", block: "nearest" })
    }
  }, [game.matchData.activeMove])
  
  const tableData: boardHistory[][] = []
  for (let i = 1; i < boardHistory.length; i += 2) {
    const rowData: boardHistory[] = []
    rowData.push(boardHistory[i])
    if (i + 1 < boardHistory.length) {
      rowData.push(boardHistory[i + 1])
    }
    tableData.push(rowData)
  }
  
  
  return (
    <div className='movesContainer' ref={tableRef}>
      <table>
        <tbody>
          {tableData.map((data, idx) => {
            return (
              <tr key={idx} className='movesRow'>
                <td>{Math.floor(idx) + 1}</td>
                <td
                  onClick={() => updateActiveState(idx * 2 + 1, game)}
                  className={game.matchData.activeMove == idx * 2 + 1 ? "highlight" : ""}
                  ref={(game.matchData.activeMove == idx * 2 + 1) || (game.matchData.activeMove == 0 && idx == 1) ? activeRef : null}
                >
                  {data[0]["algebraicNotation"]}
                </td>
                {
                  data.length > 1 ?
                    <td
                      onClick={() => updateActiveState(idx * 2 + 2, game)}
                      className={game.matchData.activeMove == idx * 2 + 2 ? "highlight" : ""}
                      ref={game.matchData.activeMove == idx * 2 + 2 ? activeRef : null}
                    >
                      {data[1]["algebraicNotation"]}
                    </td>
                    :
                    <></>
                }
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
  
}
  
function EventTypeDialog() {
  const game = useContext(GameContext)
  if (!game) {
    throw new Error("EventTypeDialog must be used within a gameContext")
  }

  const [timeoutCountdown, setTimeoutCountdown] = useState<null | number>(null)

  // Handle disconnect timeout timer
  useEffect(() => {
    const timeoutArray: NodeJS.Timeout[] = []
    const updateTime = (start: number) => {
      if (game.millisecondsUntilOpponentTimeout === null) {
        return
      }
      const deltaTime = Date.now() - start
      const timeRemainingMs = Math.max(game.millisecondsUntilOpponentTimeout - deltaTime, 0)
      setTimeoutCountdown(timeRemainingMs)
      if (timeRemainingMs == 0) {
        return
      }
      const timeoutID = setTimeout(() => {
        updateTime(start)
      }, 1000)
      timeoutArray.push(timeoutID)
    }
    if (game.millisecondsUntilOpponentTimeout != null) {
      const start = Date.now()
      const timeoutID = setTimeout(() => {
        updateTime(start)
      }, 1000)
      timeoutArray.push(timeoutID)
    }

    return () => {
      setTimeoutCountdown(null)
      for (const ID of timeoutArray) {
        clearTimeout(ID)
      }
    }
  }, [game.millisecondsUntilOpponentTimeout])



  if (game.matchData.gameOverStatus != 0) {
    return (
      <></>
    )
  }

  if (game.threefoldRepetition) {
    return (
      <div className="eventTypeDialog">
        <span>Threefold Repetition</span>
        <div>
          <button onClick={() => acceptEvent(game, OpponentEventType.ThreefoldRepetition)}>Accept</button>
          <button onClick={() => declineEvent(game, true)}>Decline</button>
        </div>
      </div>
    )
  }

  const isOpponentConnected = game.playerColour == PieceColour.White ? game.isBlackConnected : game.isWhiteConnected

  if (!isOpponentConnected && timeoutCountdown != null && timeoutCountdown <= 10_000) {

    if (timeoutCountdown > 0 ){
      return (
        <div className="eventTypeDialog">
          <span>Opponent has disconnected, can claim victory in {formatDuration(timeoutCountdown)}s</span>
        </div>
      )
    }

    return (
      <div className="eventTypeDialog">
        <span>Opponent has disconnected</span>
        <div>
          <button onClick={() => acceptEvent(game, OpponentEventType.Disconnect)}>Claim Victory</button>
        </div>
      </div>
    )
  }
  
  if (game.opponentEventType == OpponentEventType.None) {
    return <></>
  }
  
  return (
    <div className="eventTypeDialog">
      <span>Event</span>
      <div>
        <button onClick={() => acceptEvent(game, game.opponentEventType)}>Accept</button>
        <button onClick={() => declineEvent(game)}>Decline</button>
      </div>
    </div>
  )
}
  
function MoveHistoryControls() {
  
  const game = useContext(GameContext)
  
  if (!game) {
    return
  }
  
  let latestMoveButtonClassName = "moveHistoryControlsButton"
  if (game.matchData.activeMove != game.matchData.stateHistory.length - 1) {
    latestMoveButtonClassName += " newMoveNotification"
  }
  
  return (
    <div className='moveHistoryControlsContainer'>
      <div className='moveHistoryControlsButton'>
        <Microscope/>
      </ div>
      <div onClick={() => updateActiveState(0, game)} className='moveHistoryControlsButton'>
        <ChevronFirst/>
      </ div>
      <div onClick={() => updateActiveState(game?.matchData.activeMove - 1, game)} className='moveHistoryControlsButton'>
        <ChevronLeft/>
      </ div>
      <div className='moveHistoryControlsButton'>
        <ChevronRight onClick={() => updateActiveState(game?.matchData.activeMove + 1, game)}/>
      </ div>
      <div onClick={() => updateActiveState(game?.matchData.stateHistory.length - 1, game)} className={latestMoveButtonClassName}>
        <ChevronLast/>
      </ div>
      <div className='moveHistoryControlsButton'>
        <AlignJustify/>
      </ div>
    </div>
  )
}
  
function CountdownTimer({ countdownTimerMilliseconds, paused, className } : { countdownTimerMilliseconds: number, paused: boolean, className: string }) {
  // On mount, record the time using Date.now(), use a prop to get the count
  // Create a state to hold the remaining time
  // Create a function in a use effect on mount that sets the remaining time by taking
  // the elapsed time from the initial count
  //
  const [remainingTime, setRemainingTime] = useState(countdownTimerMilliseconds)
  
  useEffect(() => {
    setRemainingTime(countdownTimerMilliseconds)
    const start = Date.now()
    const updateTimer = () => {
      const delta = Date.now() - start
      setRemainingTime(countdownTimerMilliseconds - delta)
    }

    if (paused) {
      return
    }

    const intervalID = setInterval(updateTimer, 1000)

    return () => {
      clearInterval(intervalID)
    }
  }, [paused, countdownTimerMilliseconds])
  
  return (
    <div className={className} style={{backgroundColor: `${paused ? "#262421" : "#394823"}`}}>
      {formatDuration(remainingTime)}
    </div>
  )
}

function PlayerPieceCounts({ colour }: { colour: PieceColour }) {
  const game = useContext(GameContext)
  if (!game) {
    throw new Error('GameInfoTile must be used within a GameContext Provider');
  }

  const [pieceCount, setPieceCount] = useState<Map<PieceVariant, number>>(new Map<PieceVariant, number>())

  useEffect(() => {
    const newPieceCount = new Map<PieceVariant, number>()

    for (const [colour, variant] of game.matchData.activeState.board) {
      if (variant == null || colour == null) {
        continue
      }
      let count = newPieceCount.get(variant) || 0
      if (colour == PieceColour.White) {
        count += 1
      } else {
        count -= 1
      }
      newPieceCount.set(variant, count)
    }



    setPieceCount(newPieceCount)
  }, [game.matchData.activeState.board])

  // Pawns, Knights, Bishops, Rooks, Queens
  const keys = [PieceVariant.Pawn, PieceVariant.Knight, PieceVariant.Bishop, PieceVariant.Rook, PieceVariant.Queen]
  const colourMultiplier = colour == PieceColour.White ? 1 : -1
  return (
    <div className="pieceCount">
      {
        keys.map((value) => {
          const items = Array.from({ length: Math.max(0, pieceCount.get(value) as number  * colourMultiplier) }, (_, i) => (
            <div 
              key={`pieceCount-${value}-${i}`}
              className={`grey-${variantToString.get(value)}`}
              style={{ 
                position: "relative",
                display: "inline-block",
                width: `${1.7}em`,
                height: `${1.7}em`,
                backgroundSize: `${2}em`,
                backgroundPosition: "center",
                margin: "0",
              }} 
            />
          ));
          return items
        })
      }
    </div>
  )
}

export function GameInfoTile() {

  const game = useContext(GameContext)
  if (!game) {
    throw new Error('GameInfoTile must be used within a GameContext Provider');
  }

  let topTime
  let bottomTime
  let topPaused
  let bottomPaused
  let topUsername
  let bottomUsername
  let topPieceCounts
  let bottomPieceCounts

  if (game.flip) {
    topTime = game.matchData.activeState.whitePlayerTimeRemainingMilliseconds
    topPaused = isWhiteClockPaused(game)
    topUsername = game.whitePlayerUsername
    topPieceCounts = PieceColour.White

    bottomTime = game.matchData.activeState.blackPlayerTimeRemainingMilliseconds
    bottomPaused = isBlackClockPaused(game)
    bottomUsername = game.blackPlayerUsername
    bottomPieceCounts = PieceColour.Black
  } else {
    topTime = game.matchData.activeState.blackPlayerTimeRemainingMilliseconds
    topPaused = isBlackClockPaused(game)
    topUsername = game.blackPlayerUsername
    topPieceCounts = PieceColour.Black

    bottomTime = game.matchData.activeState.whitePlayerTimeRemainingMilliseconds
    bottomPaused = isWhiteClockPaused(game)
    bottomUsername = game.whitePlayerUsername
    bottomPieceCounts = PieceColour.White
  }
  
  return (
    <div className="gameInfoContainer">
      <PlayerPieceCounts colour={topPieceCounts}/>
      <CountdownTimer className="playerTimeTop" paused={topPaused} countdownTimerMilliseconds={topTime}/>
      <div className='gameInfo'>
        <EventTypeDialog />
        <PlayerInfo connected={game.isWhiteConnected} username={topUsername}/>
        <MoveHistoryControls />
        <Moves />
        <GameControls />
        <PlayerInfo connected={game.isBlackConnected} username={bottomUsername}/>
      </div>
      <CountdownTimer className="playerTimeBottom" paused={bottomPaused} countdownTimerMilliseconds={bottomTime}/>
      <PlayerPieceCounts colour={bottomPieceCounts}/>
    </div>
  )
}