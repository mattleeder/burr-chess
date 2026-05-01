import { Flame, Rabbit, TrainFront, Turtle } from 'lucide-react';
import React, { createContext, useCallback, useEffect, useReducer } from 'react';
import { API } from './api';
import './PlayerInfoTile.css';

interface PlayerInfoTilePosition {
  x: number,
  y: number,
}

export interface RatingsObject {
  bullet: number
  blitz: number
  rapid: number
  classical: number
}

export interface PlayerInfoTileData {
  playerID: number
  username: string
  pingStatus: boolean
  joinDate: number
  lastSeen: number
  ratings: RatingsObject
  numberOfGames: number
}

export interface PlayerInfoTileContextInterface {
  spawnPlayerInfoTile: (username: string, event: React.MouseEvent<HTMLElement, MouseEvent>) => void
  lightFusePlayerInfoTile: (username: string, event: React.MouseEvent<HTMLElement, MouseEvent>) => void
}

export const PlayerInfoTileContext = createContext<PlayerInfoTileContextInterface>({
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  spawnPlayerInfoTile: (_arg0, _arg1) => {return},
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  lightFusePlayerInfoTile: (_arg0, _arg1) => {return},
})

// ── State & reducer ──

const FUSE_TIMER_MS = 500

interface TileState {
  active: boolean
  username: string | null
  position: PlayerInfoTilePosition
  playerData: PlayerInfoTileData | null
  fuseActive: boolean
  queuedUsername: string | null
  queuedPosition: PlayerInfoTilePosition
}

const initialState: TileState = {
  active: false,
  username: null,
  position: { x: 0, y: 0 },
  playerData: null,
  fuseActive: false,
  queuedUsername: null,
  queuedPosition: { x: 0, y: 0 },
}

type TileAction =
  | { type: "SPAWN"; username: string; position: PlayerInfoTilePosition }
  | { type: "LIGHT_FUSE"; username: string; position: PlayerInfoTilePosition }
  | { type: "EXTINGUISH_FUSE" }
  | { type: "FUSE_EXPIRED" }
  | { type: "SET_PLAYER_DATA"; data: PlayerInfoTileData }
  | { type: "CLEAR_PLAYER_DATA" }

function isSameTile(state: TileState, username: string, position: PlayerInfoTilePosition) {
  return username === state.username && position.x === state.position.x && position.y === state.position.y
}

function tileReducer(state: TileState, action: TileAction): TileState {
  switch (action.type) {
  case "SPAWN":
    if (!state.active) {
      // Nothing active — queue tile and light fuse so it appears after delay
      return {
        ...state,
        fuseActive: true,
        queuedUsername: action.username,
        queuedPosition: action.position,
      }
    }
    if (isSameTile(state, action.username, action.position)) {
      // Same tile already shown — cancel any pending fuse
      return { ...state, fuseActive: false }
    }
    // Different tile — queue new one and light fuse to swap
    return {
      ...state,
      fuseActive: true,
      queuedUsername: action.username,
      queuedPosition: action.position,
    }

  case "LIGHT_FUSE":
    if (!state.active) {
      return { ...state, fuseActive: false }
    }
    if (isSameTile(state, action.username, action.position)) {
      // Mouse left the same element that spawned this tile — start destroy timer
      return { ...state, fuseActive: true }
    }
    // Mouse left a different element — clear queue, don't fuse
    return {
      ...state,
      fuseActive: false,
      queuedUsername: null,
      queuedPosition: { x: 0, y: 0 },
    }

  case "EXTINGUISH_FUSE":
    return { ...state, fuseActive: false }

  case "FUSE_EXPIRED":
    if (state.queuedUsername != null) {
      // Swap to queued tile
      return {
        ...state,
        active: true,
        username: state.queuedUsername,
        position: state.queuedPosition,
        playerData: null,
        fuseActive: false,
        queuedUsername: null,
        queuedPosition: { x: 0, y: 0 },
      }
    }
    // No queued tile — destroy
    return {
      ...state,
      active: false,
      fuseActive: false,
    }

  case "SET_PLAYER_DATA":
    return { ...state, playerData: action.data }

  case "CLEAR_PLAYER_DATA":
    return { ...state, playerData: null }
  }
}

// ── Helpers ──

function getPositionFromMouseEvent(event: React.MouseEvent<Element, MouseEvent>): PlayerInfoTilePosition {
  const element = event.target as HTMLElement
  const rect = element.getBoundingClientRect()
  return { x: rect.left, y: rect.top + rect.height }
}

async function fetchPlayerTileData(username: string): Promise<PlayerInfoTileData | undefined> {
  const url = API.getTileInfo + `?search=${username}`
  try {
    const response = await fetch(url, { method: "GET" })
    if (response.ok) {
      return await response.json()
    }
  } catch (e) {
    console.error(e)
  }
}

export function formatTimePassed(millisecondsSince: number) {
  const intervals: [string, number][] = [
    [" year",   31_536_000_000],
    [" month",  2_592_000_000],
    [" week",   604_800_000],
    [" day",    86_400_000],
    [" hour",   3_600_000],
    [" minute", 60_000],
    [" second", 1000],
  ]

  for (const [text, milliseconds] of intervals) {
    if (millisecondsSince >= milliseconds) {
      const multiplier = Math.floor(millisecondsSince / milliseconds)
      return `${multiplier}${text}${multiplier ? "s" : ""}`
    }
  }

  return "less than a second"
}

// ── Component ──

export function PlayerInfoTile({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(tileReducer, initialState)

  // Fetch player data when username changes
  useEffect(() => {
    if (state.username == null) {
      dispatch({ type: "CLEAR_PLAYER_DATA" })
      return
    }
    let cancelled = false
    fetchPlayerTileData(state.username).then((data) => {
      if (!cancelled && data) {
        dispatch({ type: "SET_PLAYER_DATA", data })
      }
    })
    return () => { cancelled = true }
  }, [state.username])

  // Fuse timer — fires FUSE_EXPIRED after delay
  useEffect(() => {
    if (!state.fuseActive) return
    const timeout = setTimeout(() => dispatch({ type: "FUSE_EXPIRED" }), FUSE_TIMER_MS)
    return () => clearTimeout(timeout)
  }, [state.fuseActive])

  const spawnPlayerInfoTile = useCallback((username: string, event: React.MouseEvent<HTMLElement, MouseEvent>) => {
    dispatch({ type: "SPAWN", username, position: getPositionFromMouseEvent(event) })
  }, [])

  const lightFusePlayerInfoTile = useCallback((username: string, event: React.MouseEvent<HTMLElement, MouseEvent>) => {
    dispatch({ type: "LIGHT_FUSE", username, position: getPositionFromMouseEvent(event) })
  }, [])

  const tileContext: PlayerInfoTileContextInterface = { spawnPlayerInfoTile, lightFusePlayerInfoTile }

  let timeSince: number | null = null
  if (state.playerData) {
    timeSince = Date.now() - state.playerData.joinDate * 1000
  }

  return (
    <PlayerInfoTileContext.Provider value={tileContext}>
      {children}
      {state.active &&
        <div
          className="playerInfoTile"
          style={{transform: `translate(${state.position.x}px, ${state.position.y}px)`}}
          onMouseEnter={() => dispatch({ type: "EXTINGUISH_FUSE" })}
          onMouseLeave={() => dispatch({ type: "LIGHT_FUSE", username: state.username ?? "", position: state.position })}
        >
          <div className="Name&Ping">
            {`Player Name: ${state.playerData?.username}`}
          </div>

          <div className="playerInfoTileRatings">
            <div>
              <TrainFront />
              {state.playerData?.ratings.bullet}
            </div>
            <div>
              <Flame />
              {state.playerData?.ratings.blitz}
            </div>
            <div>
              <Rabbit />
              {state.playerData?.ratings.rapid}
            </div>
            <div>
              <Turtle />
              {state.playerData?.ratings.classical}
            </div>
          </div>

          <div className="Games&JoinDate">
            <div style={{float:"left"}}>
              {`${state.playerData?.numberOfGames} games`}
            </div>
            <div style={{float:"right"}}>
              {timeSince != null ? `Joined ${formatTimePassed(timeSince)} ago` : "Unknown"}
            </div>
          </div>
        </div>
      }
    </PlayerInfoTileContext.Provider>
  )
}
