import { Search } from 'lucide-react';
import React, { useContext, useEffect, useRef, useState } from 'react';
import { PlayerInfoTileContext, PlayerInfoTileContextInterface } from './PlayerInfoTile.tsx';
import { Link } from 'react-router-dom';
import { API } from './api';
import './NavSearch.css';

interface navbarSearchResult {
  username: string
  playerID: number
  joinDate: number,
  lastSeen: number,
}

async function fetchSearchResults(searchString: string) {
  const url = API.userSearch + `?search=${searchString}`

  try {
    const response = await fetch(url, {
      method: "GET",
    })

    if (response.ok) {
      const responseData = await response.json()
      return responseData
    }

  } catch (e) {
    console.error(e)
  }
}

async function handleSearchChange(newSearchString: string, setSearchValue: React.Dispatch<React.SetStateAction<string>> ,setLoading: React.Dispatch<React.SetStateAction<boolean>>, setSearchResults: React.Dispatch<React.SetStateAction<navbarSearchResult[]>>) {
  setLoading(true)
  setSearchValue(newSearchString)
  const searchResults = await fetchSearchResults(newSearchString)
  setSearchResults(searchResults || [])
  setLoading(false)
}

function NavbarSearchInput({ ref, setLoading, setSearchResults, setSearchValue }: { ref: React.RefObject<HTMLInputElement | null>, setLoading: React.Dispatch<React.SetStateAction<boolean>>, setSearchResults: React.Dispatch<React.SetStateAction<navbarSearchResult[]>>, setSearchValue: React.Dispatch<React.SetStateAction<string>> }) {
  return (
    <input 
      className="navbarSearchInput"
      ref={ref}
      placeholder='Search'
      onChange={(event) => handleSearchChange(event.target.value, setSearchValue, setLoading, setSearchResults)}
    ></input>
  )
}

function NavbarSearchResults({ active, loading, searchResults, searchValue }: { active: boolean, loading: boolean, searchResults: navbarSearchResult[], searchValue: string }) {
  const playerInfoTile = useContext<PlayerInfoTileContextInterface>(PlayerInfoTileContext)

  if (loading || !active || searchValue == "") {
    return (
      <></>
    )
  }

  if (searchResults.length == 0 && searchValue != undefined && searchValue != "") {
    return (
      <li className="navbarSearchNoResults">No results found.</li>
    )
  }

  return (
    searchResults.map((searchResult) => {
      return (
        <li 
          className="navbarSearchResultItem" 
          key={searchResult.playerID}
          onMouseEnter={(event) => playerInfoTile?.spawnPlayerInfoTile(searchResult.username, event)}
          onMouseLeave={(event) => playerInfoTile?.lightFusePlayerInfoTile(searchResult.username, event)}
        ><Link to={`#${searchResult.playerID}`}><span>{searchResult.username}</span></Link></li>
      )
    })
  )
}

export function NavbarSearch() {
  const [searchActive, setSearchActive] = useState(false)
  const mouseOver = useRef(false)
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [loadingSearchResults, setLoadingSearchResults] = useState(false)
  const [searchResults, setSearchResults] = useState<navbarSearchResult[]>([])
  const [searchValue, setSearchValue] = useState("")



  // Close search input if mouse not over
  useEffect(() => {
    window.addEventListener("click", () => {
      if (mouseOver.current != true) {
        setSearchActive(false)
      }
    })
    return () => {
      window.removeEventListener("click", () => {
        if (!mouseOver.current != true) {
          setSearchActive(false)
        }
      })
    }
  }, [])

  useEffect(() => {
    if (searchActive && inputRef.current != null) {
      inputRef.current.focus()
    } 
  }, [searchActive])

  return (
    <div className="navbarSearchContainer" onMouseOver={() => mouseOver.current = true} onMouseOut={() => mouseOver.current = false}>
      <div className='navbarSearchResultsContainer' style={{width: `${searchActive ? "80%" : "0%"}`}}>
        <NavbarSearchInput ref={inputRef} setLoading={setLoadingSearchResults} setSearchResults={setSearchResults} setSearchValue={setSearchValue}/>
        <div className='navbarSearchResultsContent'>
          <ul>
            <NavbarSearchResults active={searchActive} loading={loadingSearchResults} searchResults={searchResults} searchValue={searchValue}/>
          </ul>
        </div>
      </div>
      <Search color='black' onClick={() => {setSearchActive(!searchActive)}}/>
    </div>
  )
}
