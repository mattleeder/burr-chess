import { ChevronLeft, ChevronRight } from 'lucide-react';
import React, { createContext, useContext, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';

// Has Top level button that is clickable
// On hover, top level button style changes and dropdown opens
// On hovering dropdown buttons, style changes
// When hover ends, revert styles and destroy dropdown

const DropdownMenuParentContext = createContext<{ menuActive: boolean, setMenuActive: (arg0: boolean) => void, parentActive: boolean, setParentActive: (arg0: boolean) => void }>({
  menuActive: false,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  setMenuActive: (_arg0: boolean) => {return},
  parentActive: false,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  setParentActive: (_arg0: boolean) => {return}
})

export function ToggleDropdownTitle({ children, active, onClick }: { children: React.ReactNode, active: boolean, onClick: () => void }) {
  const count = React.Children.count(children);

  if (count !== 1) {
    throw new Error('DropdownTitle expects exactly one child.');
  }

  let className = 'dropdownTitle'
  if (active) {
    className += " active"
  }
  
  return (
    <button className={className} onClick={() => onClick()}><span>{children}</span></button>
  )
}

function ToggleDropdownSubmenuTitle({ title, setSubmenuActive }: { title: string, setSubmenuActive: React.Dispatch<React.SetStateAction<boolean>> }) {
  const parentContext = useContext(DropdownMenuParentContext)
  
  return (
    <div style={{display: "flex"}} className='dropdownItem'
      onClick={() => {
        setSubmenuActive(false)
        parentContext.setParentActive(true)
      }}>
      <ChevronLeft />
      {title}
    </div>
  )
}

export function ToggleDropdownSubmenu({ title, children }: { title: string, children: React.ReactNode }) {
  const [submenuActive, setSubmenuActive] = useState(false)
  const parentContext = useContext(DropdownMenuParentContext)

  // Reset state on menu close
  useEffect(() => {
    if(!parentContext.menuActive) {
      setSubmenuActive(false)
    }
  }, [parentContext.menuActive])


  if (parentContext.menuActive && submenuActive) {
    return (
      <li>
        <div>
          <ToggleDropdownSubmenuTitle title={title} setSubmenuActive={setSubmenuActive} />
          <DropdownMenuParentContext.Provider value={{menuActive: parentContext.menuActive, setMenuActive: parentContext.setMenuActive, parentActive: submenuActive, setParentActive: setSubmenuActive}}>
            <ul>
              {children}
            </ul>
          </DropdownMenuParentContext.Provider>
        </div>
      </li>
    )
  }
  
  if (parentContext.menuActive && parentContext.parentActive) {
    return (
      <div style={{display: "flex"}}
        onClick={() => {
          parentContext.setParentActive(false)
          setSubmenuActive(true)
        }}
        className='dropdownItem' 
      >
        
        <li 

        >{title}</li>
        <ChevronRight style={{marginLeft: "auto"}}/>
      </div>
    )
  }

  return (
    <DropdownMenuParentContext.Provider value={{menuActive: parentContext.menuActive, setMenuActive: parentContext.setMenuActive, parentActive: submenuActive, setParentActive: setSubmenuActive}}>
      <ul>
        {children}
      </ul>
    </DropdownMenuParentContext.Provider>
  )
}

export function ToggleDropdownItem({ children, to, onClick }: { children: React.ReactNode, to?: string, onClick?: () => void }) {
  const count = React.Children.count(children);
  const parentContext = useContext(DropdownMenuParentContext)

  if (count !== 1) {
    throw new Error('DropdownItem expects exactly one child.');
  }
  // Should setParentActive to false so everything closes
  if (!parentContext.parentActive || !parentContext.menuActive) {
    return (
      <></>
    )
  }

  return (
    <li 
      className='dropdownItem' 
      onClick={() => {
        if (onClick != null) {
          onClick()
        }
        parentContext.setMenuActive(false)
        parentContext.setParentActive(false)}}
    >
      { to !== undefined ?
        <Link to={to}><span style={{width: "100%"}}>{children}</span></Link>
        :
        <span style={{width: "100%"}}>{children}</span>
      }
    </li>
  )
}

export function ToggleDropdown({ title, children, style }: { title: React.ReactNode, children: React.ReactNode, style?: React.CSSProperties }) {
  const [dropdownActive, setDropdownActive] = useState(false)
  const [menuActive, setMenuActive] = useState(false)
  const mouseOver = useRef(false)

  // Destroy dropdown if mouse not over
  useEffect(() => {
    const handler = () => {
      if (mouseOver.current != true) {
        setDropdownActive(false)
        setMenuActive(false)
      }
    }
    window.addEventListener("click", handler)
    return () => {
      window.removeEventListener("click", handler)
    }
  }, [])

  return (
    <div 
      className="dropdownContainer" 
      onMouseOver={() => {mouseOver.current = true}} 
      onMouseOut={() => {mouseOver.current = false}}
      style={style}
    >
      <ToggleDropdownTitle active={menuActive} onClick={() => {setDropdownActive(!dropdownActive); setMenuActive(!menuActive)}}>{title}</ToggleDropdownTitle>

      <DropdownMenuParentContext.Provider value={{menuActive, setMenuActive, parentActive: dropdownActive, setParentActive: setDropdownActive}}>
        <div className="dropdownContent" style={{minWidth: "200px"}}>
          <ul>
            {children}
          </ul>
        </div>
      </DropdownMenuParentContext.Provider>
    </div>
  )
}
