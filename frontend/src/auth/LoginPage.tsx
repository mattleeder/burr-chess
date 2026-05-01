import React, { useContext, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { FormError } from '../ui/forms/FormError';
import { AuthContext, LoginFormValidationErrors } from './AuthContext';
import './Auth.css';

// When redirected to login can use ?referrer=/somePage to redirect after successful login attempt

function LoginForm() {
  const [loading, setLoading] = useState(false)
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [remember, setRemember] = useState(false)
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [validationErrors, setValidationErrors] = useState<LoginFormValidationErrors>({
    username: "",
    password: "",
  })
  const auth = useContext(AuthContext)

  const handleSubmit = async (formData: FormData) => {
    if (loading) return
    setLoading(true)

    const redirectUrl = formData.get("referrer") as string || "/"
    const result = await auth.login({
      username: formData.get("username") as string,
      password: formData.get("password") as string,
      rememberMe: formData.get("rememberMe") === "true",
    })

    if (result.ok) {
      navigate(redirectUrl)
    } else if (result.data) {
      setValidationErrors(result.data)
    }

    setLoading(false)
  }

  return (
    <form action={handleSubmit}>
      <div className='formGroup'>
        <label htmlFor="username">Username</label>
        <input name="username" type="text" required={true} value={username} onChange={(event) => setUsername(event.target.value)}/>
        <FormError errorMessage={validationErrors.username} />
      </div>
      <div className='formGroup'>
        <label htmlFor="password">Password</label>
        <input name="password" type="password" required={true} value={password} onChange={(event) => setPassword(event.target.value)}/>
      </div>
      <button className='signInButton'>SIGN IN</button>
      <label>
        <input type="checkbox" name="rememberMe" style={{marginLeft: "0"}} value="true" defaultChecked={remember} onChange={(event) => setRemember(event.target.checked)}/>
        Keep me logged in
      </label>
      <input className="hidden" name="referrer" type="text" required={false} value={searchParams.get("referrer") || ""} readOnly={true}/>
    </form>
  )
}

function LoginOptions() {
  const [searchParams] = useSearchParams()
  const referrer = searchParams.get("referrer")
  let registerUrl = "/register"
  if (referrer != null) {
    registerUrl += `?referrer=${referrer}`
  }
  return (
    <div>
      <div className="loginOptions">
        <Link to={registerUrl}>Register</Link>
        <Link to="/resetPassword" style={{marginLeft: "auto"}}>Password Reset</Link>
      </div>
    </div>
  )
}

export function LoginPage() {
  return (
    <div className="loginTile">
      <h1>Sign In</h1>
      <LoginForm />
      <LoginOptions />
    </div>
  )
}
