import React, { useContext, useEffect } from "react"
import { AuthContext } from "./AuthContext"
import { useLocation, useNavigate } from "react-router-dom"

export function ProtectedRoute({ element }: { element: React.ReactNode }) {
  const auth = useContext(AuthContext)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    if (!auth.isLoading && !auth.isLoggedIn) {
      navigate(`/login?referrer=${location.pathname}`)
    }
  }, [])

  if (auth === null || auth === undefined) {
    throw new Error("ProtectedRoute must be used within an AuthContext")
  }

  if (auth.isLoading) {
    return (
      <></>
    )
  }
    
  return (
    <>
      {element}
    </>
  )
}