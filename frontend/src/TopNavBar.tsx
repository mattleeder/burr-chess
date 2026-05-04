import { useContext } from 'react';
import { Dropdown, DropdownItem } from "./Dropdown.tsx"
import { ToggleDropdown, ToggleDropdownItem, ToggleDropdownSubmenu } from './ToggleDropdown.tsx';
import { Settings } from 'lucide-react';
import { NavbarSearch } from './NavSearch.tsx';
import './TopNavBar.css';
import { LatencyDisplay } from './LatencyDisplay.tsx';
import { Link } from "react-router-dom";
import { AuthContext } from './auth/AuthContext.tsx';
import { AccountDropdown } from './auth/AccountDropdown.tsx';

export function TopNavBar() {
  const auth = useContext(AuthContext)

  return (
    <nav className='topNavBar'>
      <div className='navBarContainer left'>
        <Link to='/' className="siteName">BurrChess</Link>

        <Dropdown title='Play' titleTo='/play'>
          <DropdownItem to="#">Test</DropdownItem>
        </Dropdown>

        <Dropdown title='Watch' titleTo='/watch'>
          <DropdownItem to="/watch">All</DropdownItem>
          <DropdownItem to="/watch?timeFormat=bullet">Bullet</DropdownItem>
          <DropdownItem to="/watch?timeFormat=blitz">Blitz</DropdownItem>
          <DropdownItem to="/watch?timeFormat=rapid">Rapid</DropdownItem>
          <DropdownItem to="/watch?timeFormat=classical">Classical</DropdownItem>
        </Dropdown>
      </div>

      <div className='navBarContainer right'>
        <NavbarSearch />

        {auth.isLoggedIn ? <AccountDropdown /> : <Link to={{pathname: '/login', search: `?referrer=${window.location.pathname}`}}>Sign In</Link>}

        <ToggleDropdown title={<Settings className='navbarIcon'/>} >
          <ToggleDropdownSubmenu title="Theme">
            <ToggleDropdownItem to="#" onClick={() => console.log("Setting Theme: ThemeOne")}>ThemeOne</ToggleDropdownItem>
            <ToggleDropdownItem to="#" onClick={() => console.log("Setting Theme: ThemeOne")}>ThemeTwo</ToggleDropdownItem>
          </ToggleDropdownSubmenu>
          <ToggleDropdownSubmenu title="Language">
            <ToggleDropdownItem to="#" onClick={() => console.log("Setting Language: English")}>English</ToggleDropdownItem>
            <ToggleDropdownItem to="#" onClick={() => console.log("Setting Language: 中文")}>中文</ToggleDropdownItem>
          </ToggleDropdownSubmenu>
          <ToggleDropdownItem to="#"><LatencyDisplay /></ToggleDropdownItem>
        </ToggleDropdown>
      </div>
    </nav>
  )
}