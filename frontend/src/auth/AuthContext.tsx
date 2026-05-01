import React, { createContext, useEffect, useState } from "react";
import { submitFormData, FormResult } from "../ui/forms/FormUtilities";
import { API } from "../api";

interface AuthData {
  username: string
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
  register(data: RegisterData): Promise<FormResult<RegisterFormValidationErrors>>,
  login(data: LoginData): Promise<FormResult<LoginFormValidationErrors>>,
  logout(): Promise<FormResult>,
}

const DEFAULT_AUTH_DATA: AuthData = {
  username: "",
}

export const AuthContext = createContext<AuthContextType>({
  isLoading: true,
  isLoggedIn: false,
  authData: DEFAULT_AUTH_DATA,
  register: () => Promise.resolve({ ok: false }),
  login: () => Promise.resolve({ ok: false }),
  logout: () => Promise.resolve({ ok: false }),
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [authData, setAuthData] = useState<AuthData>(DEFAULT_AUTH_DATA)

  async function register(data: RegisterData): Promise<FormResult<RegisterFormValidationErrors>> {
    const result = await submitFormData<RegisterFormValidationErrors>(API.register, {
      credentials: "include",
      body: JSON.stringify({
        username: data.username,
        password: data.password,
        email: data.email || "",
        rememberMe: data.rememberMe,
      }),
    })

    if (result.ok && result.data) {
      setIsLoggedIn(true)
      setAuthData(result.data as unknown as AuthData)
    }

    return result
  }

  async function login(data: LoginData): Promise<FormResult<LoginFormValidationErrors>> {
    const result = await submitFormData<LoginFormValidationErrors>(API.login, {
      credentials: "include",
      body: JSON.stringify({
        username: data.username,
        password: data.password,
        rememberMe: data.rememberMe,
      }),
    })

    if (result.ok && result.data) {
      setIsLoggedIn(true)
      setAuthData(result.data as unknown as AuthData)
    }

    return result
  }

  async function logout(): Promise<FormResult> {
    const result = await submitFormData(API.logout, {
      credentials: "include",
    })

    if (result.ok) {
      setIsLoggedIn(false)
      setAuthData(DEFAULT_AUTH_DATA)
    }

    return result
  }

  useEffect(() => {
    const validateSession = async () => {
      try {
        const response = await fetch(API.validateSession, {
          method: "POST",
          credentials: "include",
          signal: AbortSignal.timeout(5000),
        })

        if (response.ok) {
          const data = await response.json()
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
    <AuthContext.Provider value={{ isLoading, isLoggedIn, authData, register, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
