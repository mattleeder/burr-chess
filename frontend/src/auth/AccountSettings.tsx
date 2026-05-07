import React, { useContext, useEffect, useState } from "react"
import { LoaderCircle } from "lucide-react"
import { FormError } from "../ui/forms/FormError"
import { submitFormData } from "../ui/forms/FormUtilities"
import { API } from "../api"
import { AuthContext } from "./AuthContext"
import "./AccountPage.css"
import { SideBar } from "../ui/SideBar"

interface AccountSettings {
  email: string | null
}

interface EmailValidationErrors {
  email: string
}

interface PasswordValidationErrors {
  currentPassword: string,
  newPassword: string,
}

async function fetchAccountSettings(signal: AbortSignal) {
  const url = API.getAccountSettings
  try {
    const response = await fetch(url, {
      signal: signal,
      method: "GET",
      credentials: "include",
    })

    if (response.ok) {
      return await response.json()
    }
  } catch (e) {
    console.error(e)
  }

  return null
}

function PasswordChange() {
  const [loading, setLoading] = useState(false)
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [validationErrors, setValidationErrors] = useState<PasswordValidationErrors>({
    currentPassword: "",
    newPassword: "",
  })
  const { csrfToken } = useContext(AuthContext)

  const handleSubmit = async (formData: FormData) => {
    if (loading) return
    setLoading(true)

    const result = await submitFormData<PasswordValidationErrors>(API.passwordChange, {
      credentials: "include",
      headers: { "X-CSRF-Token": csrfToken },
      body: JSON.stringify({
        currentPassword: formData.get("currentPassword") || "",
        newPassword: formData.get("newPassword") || "",
      }),
    })

    if (!result.ok && result.data) {
      setValidationErrors(result.data)
    }

    setLoading(false)
  }

  return (
    <form action={handleSubmit}>
      <div className='formGroup'>
        <label htmlFor="currentPassword">Current Password</label>
        <input name="currentPassword" type="password" required={true} value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)}/>
        <FormError errorMessage={validationErrors.currentPassword} />
      </div>
      <div className='formGroup'>
        <label htmlFor="newPassword">New Password</label>
        <input name="newPassword" type="password" required={true} value={newPassword} onChange={(event) => setNewPassword(event.target.value)}/>
        <FormError errorMessage={validationErrors.newPassword} />
      </div>
      <button className='signInButton'>Change Password</button>
    </form>
  )
}

function EmailChange({ accountSettings, setAccountSettings }: { accountSettings: AccountSettings, setAccountSettings: React.Dispatch<React.SetStateAction<AccountSettings | null>> }) {
  const [loading, setLoading] = useState(false)
  const [currentEmail] = useState(accountSettings.email ?? "")
  const [newEmail, setNewEmail] = useState("")
  const [validationErrors, setValidationErrors] = useState<EmailValidationErrors>({
    email: "",
  })
  const { csrfToken } = useContext(AuthContext)

  const handleSubmit = async (formData: FormData) => {
    if (loading) return
    setLoading(true)

    const result = await submitFormData<EmailValidationErrors>(API.emailChange, {
      credentials: "include",
      headers: { "X-CSRF-Token": csrfToken },
      body: JSON.stringify({
        email: formData.get("email") || "",
      }),
    })

    if (result.ok) {
      setAccountSettings((current) => {
        if (current === null) return null
        return {
          ...current,
          email: formData.get("email")?.toString() || null,
        }
      })
    } else if (result.data) {
      setValidationErrors(result.data)
    }

    setLoading(false)
  }

  return (
    <form action={handleSubmit}>
      <div className='formGroup'>
        <label htmlFor="currentEmail">Current Email</label>
        <input name="currentEmail" type="text" readOnly={true} value={currentEmail}/>
      </div>
      <div className='formGroup'>
        <label htmlFor="email">New Email</label>
        <input name="email" type="text" required={true} value={newEmail} onChange={(event) => setNewEmail(event.target.value)}/>
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
    const controller = new AbortController()

    fetchAccountSettings(controller.signal).then((data) => {
      if (!ignore) {
        setAccountSettingsData(data)
        setLoadingAccountSettings(false)
      }
    })

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
