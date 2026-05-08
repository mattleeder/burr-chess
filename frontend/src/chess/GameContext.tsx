import React, { useRef, useReducer, useCallback, useEffect, useState, createContext } from "react"
import type { ReactNode, Dispatch } from "react"
import { parseGameStateFromFEN, PieceColour, PieceVariant } from "./ChessLogic"
import { API } from "../api"

// ── Shared types ──

export interface boardInfo {
  board: [PieceColour | null, PieceVariant | null][],
  lastMove: [number, number],
  FEN: string,
  whitePlayerTimeRemainingMilliseconds: number
  blackPlayerTimeRemainingMilliseconds: number
}

export interface boardHistory {
  FEN: string,
  lastMove: [number, number]
  algebraicNotation: string
  whitePlayerTimeRemainingMilliseconds: number
  blackPlayerTimeRemainingMilliseconds: number
}

export interface matchData {
  activeState: boardInfo,
  stateHistory: boardHistory[],
  activeColour: PieceColour,
  activeMove: number,
  gameOverStatus: number,
}

export enum OpponentEventType {
  None = "none",
  Takeback = "takeback",
  Draw = "draw",
  Rematch = "rematch",
  Disconnect = "disconnect",
  Decline = "decline",
  Resign = "resign",
  ThreefoldRepetition = "threefoldRepetition"
}

// ── Context interface (unchanged for consumers) ──

export interface gameContext {
  matchData: matchData,
  setMatchData: Dispatch<React.SetStateAction<matchData>>,
  webSocket: React.RefObject<WebSocket | null>,
  playerColour: PieceColour,
  isWhiteConnected: boolean,
  isBlackConnected: boolean,
  whitePlayerUsername: string | null,
  blackPlayerUsername: string | null,
  opponentEventType: OpponentEventType,
  setOpponentEventType: Dispatch<React.SetStateAction<OpponentEventType>>,
  millisecondsUntilOpponentTimeout: number | null,
  threefoldRepetition: boolean,
  setThreefoldRepetition: Dispatch<React.SetStateAction<boolean>>,
  flip: boolean,
  setFlip: Dispatch<React.SetStateAction<boolean>>,
  wsConnectionFailed: boolean,
}

export const GameContext = createContext<gameContext | null>(null)

// ── WebSocket message types ──

interface OnConnectMessage {
  matchStateHistory: boardHistory[]
  gameOverStatus: number
  threefoldRepetition: boolean
  whitePlayerConnected: boolean
  blackPlayerConnected: boolean
  millisecondsUntilTimeout: number
  whitePlayerUsername: string | null
  blackPlayerUsername: string | null
}

interface OnMoveMessage {
  matchStateHistory: boardHistory[]
  gameOverStatus: number
  threefoldRepetition: boolean
}

interface ConnectionStatusMessage {
  playerColour: string
  isConnected: boolean
  millisecondsUntilTimeout: number,
}

interface PlayerCodeMessage {
  playerCode: number
}

interface OpponentEventMessage {
  sender: string,
  eventType: string,
}

interface ChessWebSocketMessage {
  messageType: string
  body: OnConnectMessage | OnMoveMessage | ConnectionStatusMessage | PlayerCodeMessage | OpponentEventMessage
}

// ── Reducer ──

interface GameState {
  matchData: matchData
  playerColour: PieceColour
  isWhiteConnected: boolean
  isBlackConnected: boolean
  whitePlayerUsername: string | null
  blackPlayerUsername: string | null
  millisecondsUntilOpponentTimeout: number | null
  opponentEventType: OpponentEventType
  threefoldRepetition: boolean
}

type GameAction =
  | { type: "PLAYER_CODE"; body: PlayerCodeMessage }
  | { type: "ON_CONNECT"; body: OnConnectMessage }
  | { type: "ON_MOVE"; body: OnMoveMessage }
  | { type: "CONNECTION_STATUS"; body: ConnectionStatusMessage }
  | { type: "OPPONENT_EVENT"; body: OpponentEventMessage }
  | { type: "SET_MATCH_DATA"; matchData: matchData }
  | { type: "SET_OPPONENT_EVENT_TYPE"; eventType: OpponentEventType }
  | { type: "SET_THREEFOLD_REPETITION"; value: boolean }

function applyMoveMessage(state: GameState, body: OnMoveMessage): GameState {
  const newHistory = body.matchStateHistory
  if (newHistory.length === 0) {
    console.error("New history has length 0")
    return state
  }

  const latestEntry = newHistory.at(-1) as boardHistory
  const latestFEN = latestEntry.FEN
  const activeColour = parseGameStateFromFEN(latestFEN).activeColour

  let activeState: boardInfo = {
    ...state.matchData.activeState,
    whitePlayerTimeRemainingMilliseconds: latestEntry.whitePlayerTimeRemainingMilliseconds,
    blackPlayerTimeRemainingMilliseconds: latestEntry.blackPlayerTimeRemainingMilliseconds,
  }
  let activeMove = state.matchData.activeMove

  if (state.matchData.activeState.FEN === state.matchData.stateHistory.at(-1)?.FEN) {
    activeState = {
      ...activeState,
      board: parseGameStateFromFEN(latestFEN).board,
      lastMove: latestEntry.lastMove,
      FEN: latestFEN,
    }
    activeMove = newHistory.length - 1
  }

  return {
    ...state,
    threefoldRepetition: body.threefoldRepetition,
    opponentEventType: OpponentEventType.None,
    matchData: {
      activeState,
      stateHistory: newHistory,
      activeColour,
      gameOverStatus: body.gameOverStatus,
      activeMove,
    },
  }
}

function gameReducer(state: GameState, action: GameAction): GameState {
  switch (action.type) {
  case "PLAYER_CODE":
    if (action.body.playerCode === 0) {
      return { ...state, playerColour: PieceColour.White }
    } else if (action.body.playerCode === 1) {
      return { ...state, playerColour: PieceColour.Black }
    }
    return state

  case "ON_MOVE":
    return applyMoveMessage(state, action.body)

  case "ON_CONNECT": {
    const afterMove = applyMoveMessage(state, action.body as OnMoveMessage)
    return {
      ...afterMove,
      isWhiteConnected: action.body.whitePlayerConnected,
      isBlackConnected: action.body.blackPlayerConnected,
      millisecondsUntilOpponentTimeout: action.body.millisecondsUntilTimeout > 0 ? action.body.millisecondsUntilTimeout : null,
      whitePlayerUsername: action.body.whitePlayerUsername,
      blackPlayerUsername: action.body.blackPlayerUsername,
    }
  }

  case "CONNECTION_STATUS": {
    const next = { ...state }
    if (action.body.playerColour === "white") {
      next.isWhiteConnected = action.body.isConnected
    } else if (action.body.playerColour === "black") {
      next.isBlackConnected = action.body.isConnected
    }
    next.millisecondsUntilOpponentTimeout = action.body.isConnected
      ? null
      : action.body.millisecondsUntilTimeout
    return next
  }

  case "OPPONENT_EVENT":
    switch (action.body.eventType) {
    case "takeback":
      return { ...state, opponentEventType: OpponentEventType.Takeback }
    case "draw":
      return { ...state, opponentEventType: OpponentEventType.Draw }
    case "rematch":
      return { ...state, opponentEventType: OpponentEventType.Rematch }
    }
    return state

  case "SET_MATCH_DATA":
    return { ...state, matchData: action.matchData }

  case "SET_OPPONENT_EVENT_TYPE":
    return { ...state, opponentEventType: action.eventType }

  case "SET_THREEFOLD_REPETITION":
    return { ...state, threefoldRepetition: action.value }
  }
}

// ── WebSocket message dispatcher ──

function dispatchWebSocketMessage(data: unknown, dispatch: Dispatch<GameAction>) {
  if (typeof data !== "string") return

  for (const msg of data.split("\n")) {
    let parsed: ChessWebSocketMessage
    try {
      parsed = JSON.parse(msg)
    } catch (e) {
      console.error("Failed to parse WebSocket message:", msg, e)
      continue
    }

    switch (parsed.messageType) {
    case "sendPlayerCode":
      dispatch({ type: "PLAYER_CODE", body: parsed.body as PlayerCodeMessage })
      break
    case "onConnect":
      dispatch({ type: "ON_CONNECT", body: parsed.body as OnConnectMessage })
      break
    case "connectionStatus":
      dispatch({ type: "CONNECTION_STATUS", body: parsed.body as ConnectionStatusMessage })
      break
    case "onMove":
      dispatch({ type: "ON_MOVE", body: parsed.body as OnMoveMessage })
      break
    case "opponentEvent":
      dispatch({ type: "OPPONENT_EVENT", body: parsed.body as OpponentEventMessage })
      break
    default:
      console.error("Could not understand message from websocket:", parsed.messageType)
    }
  }
}

// ── useGameWebSocket hook ──

function useGameWebSocket(matchID: string, dispatch: Dispatch<GameAction>) {
  const webSocket = useRef<WebSocket | null>(null)
  const [wsConnectionFailed, setWsConnectionFailed] = useState(false)

  useEffect(() => {
    let attempts = 0
    let cancelled = false

    const connect = () => {
      setWsConnectionFailed(false)
      webSocket.current = new WebSocket(API.matchRoom.replace("https://", "wss://") + "/" + matchID + "/ws")
      webSocket.current.onmessage = (event) => dispatchWebSocketMessage(event.data, dispatch)
      webSocket.current.onerror = (event) => console.error(event)
      webSocket.current.onclose = () => {
        if (cancelled) return
        if (attempts >= 5) {
          setWsConnectionFailed(true)
          return
        }
        attempts++
        const delay = Math.min(1000 * 2 ** attempts, 30000) // Never more than 30s
        console.log(`WebSocket closed, reconnecting in ${delay}ms (attempt ${attempts})`)
        setTimeout(connect, delay)
      }
    }
    connect()

    return () => {
      cancelled = true
      if (webSocket.current) {
        webSocket.current.onclose = null
        webSocket.current.close()
      }
    }
  }, [matchID, dispatch])

  return { webSocket, wsConnectionFailed }
}


// ── GameWrapper component ──

const INITIAL_FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

function createInitialState(timeFormatInMilliseconds: number): GameState {
  return {
    matchData: {
      activeState: {
        board: parseGameStateFromFEN(INITIAL_FEN).board,
        lastMove: [0, 0],
        FEN: INITIAL_FEN,
        whitePlayerTimeRemainingMilliseconds: timeFormatInMilliseconds,
        blackPlayerTimeRemainingMilliseconds: timeFormatInMilliseconds,
      },
      stateHistory: [{
        FEN: INITIAL_FEN,
        lastMove: [0, 0],
        algebraicNotation: "",
        whitePlayerTimeRemainingMilliseconds: timeFormatInMilliseconds,
        blackPlayerTimeRemainingMilliseconds: timeFormatInMilliseconds,
      }],
      activeColour: PieceColour.White,
      activeMove: 0,
      gameOverStatus: 0,
    },
    playerColour: PieceColour.Spectator,
    isWhiteConnected: false,
    isBlackConnected: false,
    whitePlayerUsername: null,
    blackPlayerUsername: null,
    millisecondsUntilOpponentTimeout: null,
    opponentEventType: OpponentEventType.None,
    threefoldRepetition: false,
  }
}

export function GameWrapper({ children, matchID, timeFormatInMilliseconds }: { children: ReactNode, matchID: string, timeFormatInMilliseconds: number }) {
  const [state, dispatch] = useReducer(gameReducer, timeFormatInMilliseconds, createInitialState)
  const stateRef = useRef(state)
  stateRef.current = state
  const [flip, setFlip] = useState(false)
  const { webSocket, wsConnectionFailed } = useGameWebSocket(matchID, dispatch)

  useEffect(() => {
    setFlip(state.playerColour === PieceColour.Black)
  }, [state.playerColour])

  const setMatchData = useCallback<Dispatch<React.SetStateAction<matchData>>>((action) => {
    if (typeof action === "function") {
      dispatch({ type: "SET_MATCH_DATA", matchData: action(stateRef.current.matchData) })
    } else {
      dispatch({ type: "SET_MATCH_DATA", matchData: action })
    }
  }, [])

  const setOpponentEventType = useCallback<Dispatch<React.SetStateAction<OpponentEventType>>>((action) => {
    if (typeof action === "function") {
      dispatch({ type: "SET_OPPONENT_EVENT_TYPE", eventType: action(stateRef.current.opponentEventType) })
    } else {
      dispatch({ type: "SET_OPPONENT_EVENT_TYPE", eventType: action })
    }
  }, [])

  const setThreefoldRepetition = useCallback<Dispatch<React.SetStateAction<boolean>>>((action) => {
    if (typeof action === "function") {
      dispatch({ type: "SET_THREEFOLD_REPETITION", value: action(stateRef.current.threefoldRepetition) })
    } else {
      dispatch({ type: "SET_THREEFOLD_REPETITION", value: action })
    }
  }, [])

  return (
    <GameContext.Provider value={{
      matchData: state.matchData,
      setMatchData,
      webSocket,
      playerColour: state.playerColour,
      isWhiteConnected: state.isWhiteConnected,
      isBlackConnected: state.isBlackConnected,
      whitePlayerUsername: state.whitePlayerUsername,
      blackPlayerUsername: state.blackPlayerUsername,
      opponentEventType: state.opponentEventType,
      setOpponentEventType,
      millisecondsUntilOpponentTimeout: state.millisecondsUntilOpponentTimeout,
      threefoldRepetition: state.threefoldRepetition,
      setThreefoldRepetition,
      flip,
      setFlip,
      wsConnectionFailed,
    }}>
      {children}
    </GameContext.Provider>
  )
}
