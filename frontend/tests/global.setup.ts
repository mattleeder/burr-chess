async function globalSetup() {
  const backendURL = process.env.BACKEND_URL ?? 'http://localhost:8080'
  try {
    await fetch(`${backendURL}/resetQueues`)
    await fetch(`${backendURL}/resetLiveMatches`)
    await fetch(`${backendURL}/resetMatchClients`)
    await fetch(`${backendURL}/resetRateLimiters`)
  } catch (e) {
    console.warn('Could not reset backend state (backend may not be running):', e)
  }
}

export default globalSetup
