import { useContext, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { FormError } from '../ui/forms/FormError';
import { AuthContext, RegisterFormValidationErrors } from './AuthContext';
import './Auth.css';

// When redirected to login can use ?referrer=/somePage to redirect after successful login attempt

function RegisterForm() {
  const [loading, setLoading] = useState(false)
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [email, setEmail] = useState("")
  const [remember, setRemember] = useState(false)
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [validationErrors, setValidationErrors] = useState<RegisterFormValidationErrors>({
    username: "",
    password: "",
    email: ""
  })
  const auth = useContext(AuthContext)

  const handleSubmit = async (formData: FormData) => {
    if (loading) return
    setLoading(true)

    const redirectUrl = formData.get("referrer") as string || "/"
    const result = await auth.register({
      username: formData.get("username") as string,
      password: formData.get("password") as string,
      email: formData.get("email") as string || undefined,
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
        <input name="username" type="text" required={true} value={username} onChange={(event) => setUsername(event.target.value)} />
        <FormError errorMessage={validationErrors.username} />
      </div>
      <div className='formGroup'>
        <label htmlFor="password">Password</label>
        <input name="password" type="password" required={true} value={password} onChange={(event) => setPassword(event.target.value)}/>
      </div>
      <div className='formGroup'>
        <label htmlFor="email">Email (Optional - For password reset)</label>
        <input name="email" type="email" required={false} value={email} onChange={(event) => setEmail(event.target.value)}/>
      </div>
      <button className={`signInButton${loading ? " disabled" : ""}`}>REGISTER</button>
      <label>
        <input type="checkbox" style={{marginLeft: "0"}} checked={remember} onChange={() => setRemember(!remember)}/>
        Keep me logged in
      </label>
      <input className="hidden" name="referrer" type="text" required={false} value={searchParams.get("referrer") || ""}/>
    </form>
  )
}

export function RegisterPage() {
  return (
    <div className="registerTile">
      <h1>Register</h1>
      <RegisterForm />
    </div>
  )
}
