package main

import "math"

func getKFactor(elo int64) float64 {
	if elo < 2100 {
		return 32
	} else if elo <= 2400 {
		return 24
	}
	return 16
}

func calculateEloChanges(playerOneElo int64, playerOnePoints float64, playerTwoElo int64, playerTwoPoints float64) (playerOneEloGain float64, playerTwoEloGain float64) {
	playerOneExpectedPoints := 1.0 / (1 + math.Pow(10, float64(playerTwoElo-playerOneElo)/400))
	playerTwoExpectedPoints := 1.0 / (1 + math.Pow(10, float64(playerOneElo-playerTwoElo)/400))

	playerOneEloGain = getKFactor(playerOneElo) * (playerOnePoints - playerOneExpectedPoints)
	playerTwoEloGain = getKFactor(playerTwoElo) * (playerTwoPoints - playerTwoExpectedPoints)

	return playerOneEloGain, playerTwoEloGain
}
