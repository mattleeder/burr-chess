import { useContext } from 'react';
import { CircleUserRound, LogOut } from 'lucide-react';
import { ToggleDropdown, ToggleDropdownItem } from '../ToggleDropdown.tsx';
import { AuthContext } from './AuthContext.tsx';

export function AccountDropdown() {
  const auth = useContext(AuthContext)

  return (
    <ToggleDropdown title={<CircleUserRound className='navbarIcon'/>} >
      <ToggleDropdownItem to={`/user/${auth.authData.username}`}>Profile</ToggleDropdownItem>
      <ToggleDropdownItem to={"account/settings"}>Account Settings</ToggleDropdownItem>
      <ToggleDropdownItem onClick={() => auth.logout()}><span style={{display: "flex", width: "100%"}}><span>Log Out</span><LogOut style={{marginLeft: "auto"}}/></span></ToggleDropdownItem>
    </ToggleDropdown>
  )
}