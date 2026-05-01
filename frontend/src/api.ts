const BASE_URL = import.meta.env.VITE_API_URL as string

export const API = {
  fetchMoves:         `${BASE_URL}/getMoves`,
  makeMove:           `${BASE_URL}/makeMove`,
  joinQueue:          `${BASE_URL}/joinQueue`,
  listenForMatch:     `${BASE_URL}/listenformatch`,
  matchRoom:          `${BASE_URL}/matchroom`,
  fetchHighestElo:    `${BASE_URL}/getHighestEloMatch`,
  register:           `${BASE_URL}/register`,
  login:              `${BASE_URL}/login`,
  logout:             `${BASE_URL}/logout`,
  validateSession:    `${BASE_URL}/validateSession`,
  userSearch:         `${BASE_URL}/userSearch`,
  getTileInfo:        `${BASE_URL}/getTileInfo`,
  getPastMatches:     `${BASE_URL}/getPastMatches`,
  getAccountSettings: `${BASE_URL}/getAccountSettings`,
  passwordChange:     `${BASE_URL}/passwordChange`,
  emailChange:        `${BASE_URL}/emailChange`,
} as const
