import React, { createContext, useEffect, useState } from "react";
import { submitFormData, FormResult } from "../ui/forms/FormUtilities";
import { API } from "../api";

interface AuthData {
  username: string
  csrfToken: string
}

interface RegisterData {
  username: string
  password: string
  email?: string
  rememberMe: boolean
}

export interface RegisterFormValidationErrors {
  username: string
  password: string
  email: string
}

interface LoginData {
  username: string
  password: string
  rememberMe: boolean
}

export interface LoginFormValidationErrors {
  username: string
  password: string
}

export interface AuthContextType {
  isLoading: boolean,
  isLoggedIn: boolean,
  authData: AuthData,
  csrfToken: string,
  register(data: RegisterData): Promise<FormResult<RegisterFormValidationErrors>>,
  login(data: LoginData): Promise<FormResult<LoginFormValidationErrors>>,
  logout(): Promise<FormResult>,
}

const DEFAULT_AUTH_DATA: AuthData = {
  username: "",
  csrfToken: "",
}

export const AuthContext = createContext<AuthContextType>({
  isLoading: true,
  isLoggedIn: false,
  authData: DEFAULT_AUTH_DATA,
  csrfToken: "",
  register: () => Promise.resolve({ ok: false }),
  login: () => Promise.resolve({ ok: false }),
  logout: () => Promise.resolve({ ok: false }),
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [authData, setAuthData] = useState<AuthData>(DEFAULT_AUTH_DATA)
  const [csrfToken, setCsrfToken] = useState("")

  async function register(data: RegisterData): Promise<FormResult<RegisterFormValidationErrors>> {
    if (!csrfToken) return { ok: false }
    const result = await submitFormData<RegisterFormValidationErrors>(API.register, {
      credentials: "include",
      headers: { "X-CSRF-Token": csrfToken },
      body: JSON.stringify({
        username: data.username,
        password: data.password,
        email: data.email || "",
        rememberMe: data.rememberMe,
      }),
    })

    if (result.ok && result.data) {
      const auth = result.data as unknown as AuthData
      setIsLoggedIn(true)
      setAuthData(auth)
      setCsrfToken(auth.csrfToken)
    }

    return result
  }

  async function login(data: LoginData): Promise<FormResult<LoginFormValidationErrors>> {
    if (!csrfToken) return { ok: false }
    const result = await submitFormData<LoginFormValidationErrors>(API.login, {
      credentials: "include",
      headers: { "X-CSRF-Token": csrfToken },
      body: JSON.stringify({
        username: data.username,
        password: data.password,
        rememberMe: data.rememberMe,
      }),
    })

    if (result.ok && result.data) {
      const auth = result.data as unknown as AuthData
      setIsLoggedIn(true)
      setAuthData(auth)
      setCsrfToken(auth.csrfToken)
    }

    return result
  }

  async function logout(): Promise<FormResult> {
    const result = await submitFormData(API.logout, {
      credentials: "include",
      headers: { "X-CSRF-Token": csrfToken },
    })

    if (result.ok) {
      setIsLoggedIn(false)
      setAuthData(DEFAULT_AUTH_DATA)
    }

    return result
  }

  // Signal to E2E tests that auth is initialised and csrfToken is ready.
  // Runs after React commits the render where isLoading becomes false, which
  // guarantees csrfToken is also committed (they update in the same batch).
  useEffect(() => {
    if (!isLoading) {
      document.body.setAttribute('data-auth-loaded', 'true')
    }
  }, [isLoading])

  useEffect(() => {
    const validateSession = async () => {
      try {
        const response = await fetch(API.validateSession, {
          method: "POST",
          credentials: "include",
          signal: AbortSignal.timeout(5000),
        })

        // 401 returns JSON with csrfToken but no username, so logged-out
        // users still get a CSRF token for protected requests like joinQueue.
        const data = await response.json()
        setCsrfToken(data.csrfToken)
        if (response.ok) {
          setIsLoggedIn(true)
          setAuthData(data)
        }
      } catch (e) {
        console.error(e)
      } finally {
        setIsLoading(false)
      }
    }

    validateSession()
  }, [])

  return (
    <AuthContext.Provider value={{ isLoading, isLoggedIn, authData, csrfToken, register, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
