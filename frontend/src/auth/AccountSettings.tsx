import React, { useEffect, useState } from "react"
import { LoaderCircle } from "lucide-react"
import { FormError } from "../ui/forms/FormError"
import { submitFormData } from "../ui/forms/FormUtilities"
import { SQLNullString } from "../chess/GameContext"
import { SideBar } from "../ui/SideBar"

interface AccountSettings {
  email: SQLNullString
}

interface EmailValidationErrors {
  email: string
}

interface PasswordValidationErrors {
  currentPassword: string,
  newPassword: string,
}

async function fetchAccountSettings(signal: AbortSignal) {
  const url = import.meta.env.VITE_API_GET_ACCOUNT_SETTINGS
  try {
    const response = await fetch(url, {
      signal: signal,
      method: "GET",
      credentials: "include",
    })

    if (response.ok) {
      const data = await response.json()
      return data
    }
  } catch (e) {
    console.error(e)
  }

  return null
}

async function handlePasswordChange(
  formData: FormData,
  setLoading: React.Dispatch<React.SetStateAction<boolean>>,
  setValidationErrors: React.Dispatch<React.SetStateAction<PasswordValidationErrors>>) {
  const url = import.meta.env.VITE_API_PASSWORD_CHANGE_URL
  const options: RequestInit = {
    credentials: "include",
    body: JSON.stringify({
      currentPassword: formData.get("password") || "",
      newPassword: formData.get("password") || "",
    })
  }

  const passwordCallback = (success: boolean, responseData: unknown) => {
    if (success) {
      console.log("GOOD")
    } else {
      setValidationErrors(responseData as PasswordValidationErrors)
    }
  }

  submitFormData(url, options, passwordCallback, setLoading)
}

function PasswordChange() {
  const [loading, setLoading] = useState(false)
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [validationErrors, setValidationErrors] = useState<PasswordValidationErrors>({
    currentPassword: "",
    newPassword: "",
  })
  return (
    <form action={(formData) => {if (!loading) {handlePasswordChange(formData, setLoading, setValidationErrors)}}}>
      <div className='formGroup'>
        <label htmlFor="password">Current Password</label>
        <input name="currentPassword" type="password" required={true} value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)}/>
        <FormError errorMessage={validationErrors.currentPassword} />
      </div>
      <div className='formGroup'>
        <label htmlFor="password">New Password</label>
        <input name="newPassword" type="password" required={true} value={newPassword} onChange={(event) => setNewPassword(event.target.value)}/>
        <FormError errorMessage={validationErrors.newPassword} />
      </div>
      <button className='signInButton'>Change Password</button>
    </form>
  )
}

async function handleEmailChange(
  formData: FormData,
  setLoading: React.Dispatch<React.SetStateAction<boolean>>,
  setAccountSettings: React.Dispatch<React.SetStateAction<AccountSettings | null>>,
  setValidationErrors: React.Dispatch<React.SetStateAction<EmailValidationErrors>>) {
  const url = import.meta.env.VITE_API_EMAIL_CHANGE_URL
  const options: RequestInit = {
    credentials: "include",
    body: JSON.stringify({
      email: formData.get("email") || "",
    })
  }

  const emailCallback = (success: boolean, responseData: unknown) => {
    if (success) {
      setAccountSettings((currentSettings) => {
        if (currentSettings !== null) {
          const newSettings = {...currentSettings}
          newSettings.email = {
            Valid: true,
            String: formData.get("email")?.toString() || ""
          }
        }
        return null
      })
    } else {
      setValidationErrors(responseData as EmailValidationErrors)
    }
  }

  submitFormData(url, options, emailCallback, setLoading)
}

function EmailChange({ accountSettings, setAccountSettings }: { accountSettings: AccountSettings, setAccountSettings: React.Dispatch<React.SetStateAction<AccountSettings | null>> }) {
  const [loading, setLoading] = useState(false)
  const [currentEmail, _setCurrentEmail] = useState(accountSettings.email.Valid ? accountSettings.email.String : "")
  const [newEmail, setNewEmail] = useState("")
  const [validationErrors, setValidationErrors] = useState<EmailValidationErrors>({
    email: "",
  })
  return (
    <form action={(formData) => {if (!loading) {handleEmailChange(formData, setLoading, setAccountSettings, setValidationErrors)}}}>
      <div className='formGroup'>
        <label htmlFor="currentEmail">Current Email</label>
        <input name="currentEmail" type="text" readOnly={true} value={currentEmail}/>
      </div>
      <div className='formGroup'>
        <label htmlFor="newEmail">New Email</label>
        <input name="newEmail" type="text" required={true} value={newEmail} onChange={(event) => setNewEmail(event.target.value)}/>
        <FormError errorMessage={validationErrors.email} />
      </div>
      <button className='signInButton'>Change Email</button>
      <button className='signInButton' style={{backgroundColor: "red"}}>Remove Email</button>
    </form>
  )
}

export function AccountSettingsPage() {
  const [loadingAccountSettings, setLoadingAccountSettings] = useState(true)
  const [accountSettingsData, setAccountSettingsData] = useState<AccountSettings | null>(null)

  useEffect(() => {
    let ignore = false
    setLoadingAccountSettings(true)
    const controller = new AbortController()
    const signal = controller.signal;

    (async() => {
      const accountSettingsData = fetchAccountSettings(signal)
      if (!ignore) {
        setAccountSettingsData(await accountSettingsData)
        setLoadingAccountSettings(false)
      }}
    )()

    return () => {
      ignore = true
      controller.abort("page change")
    }
  }, [])

  if (loadingAccountSettings) {
    return (
      <div>
        <LoaderCircle className="loaderSpin"/>
      </div>
    )
  }

  if (accountSettingsData === null) {
    return (
      <div>
        Error getting data.
      </div>
    )
  }

  return (
    <div>
      <SideBar tabs={[
        {title: "Change Email", content: <EmailChange accountSettings={accountSettingsData} setAccountSettings={setAccountSettingsData}/>},
        {title: "Change Password", content: <PasswordChange />},
      ]}/>
    </div>
  )
}