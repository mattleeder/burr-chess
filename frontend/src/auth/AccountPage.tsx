import React, { useEffect, useRef, useState } from "react"
import { formatTimePassed, PlayerInfoTileData } from "../PlayerInfoTile"
import { Flame, LoaderCircle, Rabbit, TrainFront, Turtle } from "lucide-react"
import { useParams } from "react-router-dom"
import { pastMatchData, MatchTile } from "../WatchPage"
import { PingStatus } from "../chess/GameInfoTile"
import { API } from "../api"
import "./AccountPage.css"

enum Page {
  All = "all",
  Bullet = "bullet",
  Blitz = "blitz",
  Rapid = "rapid",
  Classical = "classical",
}

function AccountInfoDisplay({ accountInfo }: { accountInfo: PlayerInfoTileData | undefined }) {

  if (accountInfo === undefined) {
    return (
      <div>No data.</div>
    )
  }

  return (
    <div style={{
      position: "relative",
      width: "70vw",
      backgroundColor: "#ababaa",
      borderRadius: "4px",
      padding: "1em",
      marginLeft: "auto",
      marginRight: "auto",
      
    }}>
      <div style={{display: "flex", flexDirection: "row"}}>
        <div style={{display: "flex", flexDirection: "row", alignItems: "center"}}>
          <PingStatus style={{width: "1em", paddingRight: "0.5em"}} connected={accountInfo.pingStatus}/>
          <span>{accountInfo.username}</span>
        </div>
        <div style={{
          display: "grid",
          gridTemplateColumns: "auto auto",
          columnGap: "1em",
          marginLeft: "auto",
          alignItems: "start",
          textAlign: "left",
        }}>
          <div>
            Joined:
          </div>
          <div>
            {formatTimePassed(Date.now() - accountInfo.joinDate * 1000)} ago
          </div>
          <div>
            Last Seen:
          </div>
          <div>
            {formatTimePassed(Date.now() - accountInfo.lastSeen * 1000)} ago  
          </div>
          <div>
            Number of Games:
          </div>
          <div>
            {accountInfo.numberOfGames}
          </div>
        </div>
      </div>
    </div>
  )
}


function AccountContent({ pageData }: { pageData: pastMatchData[] | undefined }) {

  if (pageData === undefined || pageData === null) {
    return (
      <div>
        No data.
      </div>
    )
  }
  return (
    <div style={{
      marginLeft: "auto",
      marginRight: "auto",
    }}>
      <ul>
        {pageData.map((pastMatchData, idx) => {
          return (
            <MatchTile key={`match_${pastMatchData.matchID}`} matchData={pastMatchData} idx={idx}/>
          )
        })}
      </ul>
    </div>
  )
}

interface FetchResult<T> {
  data?: T
  error: boolean
}

async function fetchPageData(username: string, activePage: Page, pageCache: React.RefObject<Map<Page, pastMatchData[]>>, signal: AbortSignal): Promise<FetchResult<pastMatchData[]>> {
  let url = API.getPastMatches
  const searchParams: [string, string][] = []

  if (username != "") {
    searchParams.push(["username", username])
  }

  if (activePage != Page.All) {
    searchParams.push(["timeFormat", activePage])
  }

  if (searchParams.length > 0) {
    url += "?"
  }

  for (const [searchTerm, value] of searchParams) {
    url += `${searchTerm}=${value}&`
  }

  try {
    const response = await fetch(url, { signal, method: "GET" })

    if (response.ok) {
      const data: pastMatchData[] = await response.json()
      pageCache.current?.set(activePage, data)
      return { data, error: false }
    }
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") return { error: false }
    console.error(e)
  }

  return { error: true }
}

async function fetchUserData(username: string, signal: AbortSignal) {
  const url = API.getTileInfo + `?search=${username}`

  try {
    const response = await fetch(url, {
      signal: signal,
      method: "GET",
    })

    if (response.ok) {
      const responseData: PlayerInfoTileData = await response.json()
      return responseData
    }

  } catch (e) {
    console.error(e)
  }

  return undefined
}

export function AccountPage() {
  const { username } = useParams()
  const [activePage, setActivePage] = useState(Page.All)
  const [pageData, setPageData] = useState<pastMatchData[] | undefined>(undefined)
  const [pageError, setPageError] = useState(false)
  const [playerData, setPlayerData] = useState<PlayerInfoTileData | undefined>(undefined)
  const pageCache = useRef(new Map<Page, pastMatchData[]>())
  const [loadingPlayerData, setLoadingPlayerData] = useState(true)
  const [loadingContent, setLoadingContent] = useState(true)

  // Get user data
  useEffect(() => {
    let ignore = false
    setLoadingPlayerData(true)
    const controller = new AbortController()
    const signal = controller.signal;

    (async() => {
      const playerData = fetchUserData(username || "", signal)
      if (!ignore) {
        setPlayerData(await playerData)
        setLoadingPlayerData(false)
      }}
    )()

    return () => {
      ignore = true
      controller.abort("username changed")
    }
  }, [username])

  // Get Page Content
  useEffect(() => {
    let ignore = false
    setLoadingContent(true)
    const controller = new AbortController()
    const signal = controller.signal;

    (async() => {
      const cached = pageCache.current?.get(activePage)
      if (cached) {
        if (!ignore) {
          setPageData(cached)
          setPageError(false)
          setLoadingContent(false)
        }
        return
      }
      const result = await fetchPageData(username || "", activePage, pageCache, signal)
      if (!ignore) {
        setPageData(result.data)
        setPageError(result.error)
        setLoadingContent(false)
      }
    })()

    return () => {
      ignore = true
      controller.abort("activePage changed")
    }
  }, [username, activePage])

  return (
    <>
      <div style={{
        display: "block",
        position: "fixed",
        left: 0,
        top: "3em",
        textAlign: "left",
        padding: 0,
        margin: 0,
      }}>
        <ul style={{
          display: "block",
          listStyle: "none",
          padding: 0,
          margin: 0,
        }}>
          <li className={`userTimeFormat${activePage == Page.All ? " active" : ""}`} onClick={() => setActivePage(Page.All)}>
            <div>
              <span>All</span>
            </div>
          </li>
          <li className={`userTimeFormat${activePage == Page.Bullet ? " active" : ""}`} onClick={() => setActivePage(Page.Bullet)}>
            <div style={{display: "flex", flexDirection: "column", width: "100%"}}>
              <div style={{display: "flex", justifyItems: "center"}}>
                <span>Bullet</span>
                <TrainFront style={{marginLeft: "auto"}}/>
              </div>
              <div style={{textAlign: "center"}}>
                {playerData?.ratings.bullet || "-"}
              </div>
            </div>
          </li>
          <li className={`userTimeFormat${activePage == Page.Blitz ? " active" : ""}`} onClick={() => setActivePage(Page.Blitz)}>
            <div style={{display: "flex", flexDirection: "column", width: "100%"}}>
              <div style={{display: "flex", justifyItems: "center"}}>
                <span>Blitz</span>
                <Flame style={{marginLeft: "auto"}}/>
              </div>
              <div style={{textAlign: "center"}}>
                {playerData?.ratings.blitz || "-"}
              </div>
            </div>
          </li>
          <li className={`userTimeFormat${activePage == Page.Rapid ? " active" : ""}`} onClick={() => setActivePage(Page.Rapid)}>
            <div style={{display: "flex", flexDirection: "column", width: "100%"}}>
              <div style={{display: "flex", justifyItems: "center"}}>
                <span>Rapid</span>
                <Rabbit style={{marginLeft: "auto"}}/>
              </div>
              <div style={{textAlign: "center"}}>
                {playerData?.ratings.rapid || "-"}
              </div>
            </div>
          </li>
          <li className={`userTimeFormat${activePage == Page.Classical ? " active" : ""}`} onClick={() => setActivePage(Page.Classical)}>
            <div style={{display: "flex", flexDirection: "column", width: "100%"}}>
              <div style={{display: "flex", justifyItems: "center"}}>
                <span>Classical</span>
                <Turtle style={{marginLeft: "auto"}}/>
              </div>
              <div style={{textAlign: "center"}}>
                {playerData?.ratings.classical || "-"}
              </div>
            </div>
          </li>
        </ul>
      </div>
      <div style={{
        display:"flex",
        flexDirection: "column",
        height: "100vh",
        paddingTop: "3em",
        marginLeft: "auto",
        marginRight: "auto",
      }}>
        {loadingPlayerData ? <LoaderCircle className="loaderSpin"/> : <AccountInfoDisplay accountInfo={playerData}/>}
        {loadingContent ? <LoaderCircle className="loaderSpin"/> : pageError ? <div>Failed to load matches.</div> : <AccountContent pageData={pageData}/>}
      </div>
    </>
  )
}