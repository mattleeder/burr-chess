import React, { useContext, useEffect, useState } from "react";
import { LoaderCircle, Swords, Flame, Rabbit, TrainFront, Turtle } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { FrozenChessBoard } from "./chess/ChessBoard";
import { parseGameStateFromFEN } from "./chess/ChessLogic";
import { PlayerInfoTileContext, PlayerInfoTileContextInterface } from "./PlayerInfoTile";
import { SQLNullString, SQLNullInt64 } from "./types";
import { API } from "./api";
import "./WatchPage.css";

const resultReasons = [
  "Ongoing",
  "Stalemate",
  "Checkmate",
  "ThreefoldRepetition",
  "InsufficientMaterial",
  "WhiteFlagged",
  "BlackFlagged",
  "Draw",
  "WhiteResigned",
  "BlackResigned",
  "Abort",
  "WhiteDisconnected",
  "BlackDisconnected",
]

export interface pastMatchData {
  matchID: number
  whitePlayerUsername: SQLNullString
  blackPlayerUsername: SQLNullString
  lastMovePiece: SQLNullInt64
  lastMoveMove: SQLNullInt64
  finalFEN: string
  timeFormatInMilliseconds: number
  incrementInMilliseconds: number
  result: number
  resultReason: number
  whitePlayerElo: number
  blackPlayerElo: number
  whitePlayerEloGain: number
  blackPlayerEloGain: number
  matchStartTime: number
  matchEndTime: number
  averageElo: number
}

function FormatIcon({ timeFormatInMilliseconds, style }: { timeFormatInMilliseconds: number, style?: React.CSSProperties }) {
  const minute = 60_000
  if (timeFormatInMilliseconds < 2 * minute) {
    return <TrainFront style={style}/>
  } else if (timeFormatInMilliseconds < 5 * minute) {
    return <Flame style={style}/>
  } else if (timeFormatInMilliseconds < 20 * minute) {
    return <Rabbit style={style}/>
  } else {
    return <Turtle style={style}/>
  }
}

interface FetchMatchesResult {
  matches: pastMatchData[]
  error: boolean
}

async function fetchMatches(searchParams: URLSearchParams, signal: AbortSignal): Promise<FetchMatchesResult> {
  const timeFormat = searchParams.get("timeFormat") || ""
  const url = API.getPastMatches + `?timeFormat=${timeFormat}`

  try {
    const response = await fetch(url, { signal, method: "GET" })

    if (response.ok) {
      const matches: pastMatchData[] = await response.json()
      return { matches, error: false }
    }
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") return { matches: [], error: false }
    console.error(e)
  }

  return { matches: [], error: true }
}

function getTimeFormatName(timeFormatInMilliseconds: number) {
  const minute = 60_000
  if (timeFormatInMilliseconds < 2 * minute) {
    return "Bullet"
  } else if (timeFormatInMilliseconds < 5 * minute) {
    return "Blitz"
  } else if (timeFormatInMilliseconds < 20 * minute) {
    return "Rapid"
  } else {
    return "Classical"
  }
}

export function MatchTile({ matchData, idx }: { matchData: pastMatchData, idx: number }) {
  const playerInfoTile = useContext<PlayerInfoTileContextInterface>(PlayerInfoTileContext)
  let outcome = ""
  if (matchData.result == 0) {
    outcome = "White wins"
  } else if (matchData.result == 1) {
    outcome = "Black wins"
  } else {
    outcome = "Draw"
  }

  outcome += ` by ${resultReasons[matchData.resultReason]}`

  const gameState = parseGameStateFromFEN(matchData.finalFEN)
  let liClassname = ""
  if (idx % 2 == 0) {
    liClassname += "even"
  } else {
    liClassname += "odd"
  }
  liClassname += " matchRow"

  return (
    <li className={liClassname}>
      <Link to={`/matchroom/${matchData.matchID}`} style={{display: "block", position: "absolute", width: "100%", height: "100%", boxSizing: "content-box", zIndex: "3"}}/>
      <div style={{marginTop: "auto", marginBottom: "auto", boxShadow: "2px 2px 2px #000000", height: "inherit", width: "35vh"}}>
        {/* Chessboard, display final position */}
        <FrozenChessBoard board={gameState.board} lastMove={[matchData.lastMovePiece.Int64, matchData.lastMoveMove.Int64]} showLastMove={matchData.lastMovePiece.Valid && matchData.lastMoveMove.Valid}/>
      </div>
      <div style={{display: "grid", gridTemplateRows: "1fr 1fr", marginLeft: "1em", height: "inherit"}}>
        {/* Info, grid 2 rows, top row is format info and date, 2nd row is player Info and victory */}
        <div style={{display: "grid", gridTemplateRows: "1fr 1fr"}}>
          {/* Grid 2 columns, first column is icon for rating, 2nd is info */}
          <div>
            {/* Icon for rating */}
            <FormatIcon timeFormatInMilliseconds={matchData.timeFormatInMilliseconds} style={{float: "left"}}/>
            <span style={{float: "left"}}>{Math.floor(matchData.timeFormatInMilliseconds / 60_000)}+{Math.floor(matchData.incrementInMilliseconds / 1000)} • {getTimeFormatName(matchData.timeFormatInMilliseconds)}</span>
          </div>
          <div style={{display: "grid", gridTemplateRows: "1fr 1fr"}}>
            {/* Info, grid 2 rows, top is rating info bottom is date */}
            <div>
              <span style={{float: "left"}}>{`${new Date(matchData.matchEndTime * 1000).toLocaleString()}`}</span>
            </div>
          </div>
        </div>
        <div style={{display: "grid", gridTemplateRows: "1fr 1fr"}}>
          {/* Player Info, grid 2 rows, top is player info, bottom is game outcome */}
          <div style={{display: "grid", gridTemplateColumns: "1fr 1fr 1fr"}}>
            {/* Player info, grid 3 columns, 1st is white info, 2nd is vs icon, 3rd is black info */}
            <div>
              {/* White info*/}
              {matchData.whitePlayerUsername.Valid ? 
                <span style={{position:"relative", zIndex: "4"}}
                  onMouseEnter={(event) => playerInfoTile?.spawnPlayerInfoTile(matchData.whitePlayerUsername.String, event)}
                  onMouseLeave={(event) => playerInfoTile?.lightFusePlayerInfoTile(matchData.whitePlayerUsername.String, event)}
                >
                  {`${matchData.whitePlayerUsername.String} (${matchData.whitePlayerElo} ${matchData.whitePlayerEloGain >= 0 ? "+" : ""}${matchData.whitePlayerEloGain})`}
                </span>
                : 
                <span>Anon</span>}
            </div>
            <div>
              {/* VS icon */}
              <Swords />
            </div>
            <div>
              {/* Black info */}
              {matchData.blackPlayerUsername.Valid ? 
                <span style={{position:"relative", zIndex: "4"}}
                  onMouseEnter={(event) => playerInfoTile?.spawnPlayerInfoTile(matchData.blackPlayerUsername.String, event)}
                  onMouseLeave={(event) => playerInfoTile?.lightFusePlayerInfoTile(matchData.blackPlayerUsername.String, event)}
                >
                  {`${matchData.blackPlayerUsername.String} (${matchData.blackPlayerElo} ${matchData.blackPlayerEloGain >= 0 ? "+" : ""}${matchData.blackPlayerEloGain})`}
                </span>
                : 
                <span>Anon</span>}
            </div>
          </div>
          <div>
            {outcome}
          </div>
        </div>
      </div>
    </li>
  )
}

export function WatchPage() {
  const [fetchingMatches, setFetchingMatches] = useState(true)
  const [fetchError, setFetchError] = useState(false)
  const [searchParams, _setSearchParams] = useSearchParams()
  const [matchList, setMatchList] = useState<pastMatchData[]>([])

  useEffect(() => {
    const controller = new AbortController()
    const signal = controller.signal
    const getMatchList = async () => {
      const result = await fetchMatches(searchParams, signal)
      setMatchList(result.matches)
      setFetchError(result.error)
      setFetchingMatches(false)
    }
    getMatchList()
    return () => {
      controller.abort("WatchPage searchParam changed")
    }
  }, [searchParams])

  if (fetchingMatches) {
    return (
      <div>
        <LoaderCircle className="loaderSpin"/>
      </div>
    )
  }

  if (fetchError) {
    return (
      <div>Failed to load matches.</div>
    )
  }

  return (
    <div>
      <ul>
        {matchList.map((matchData, idx) => {
          return (
            <MatchTile key={`match_${matchData.matchID}`} matchData={matchData} idx={idx}/>
          )
        })}
      </ul>
    </div>
  )
}