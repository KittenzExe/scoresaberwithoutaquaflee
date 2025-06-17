package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type PlayersResponse struct {
	Players []Player `json:"players"`
}

type Player struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	ProfilePicture string      `json:"profilePicture"`
	Bio            *string     `json:"bio"`
	Country        string      `json:"country"`
	PP             float64     `json:"pp"`
	Rank           int         `json:"rank"`
	CountryRank    int         `json:"countryRank"`
	Role           *string     `json:"role"`
	Badges         interface{} `json:"badges"`
	Histories      string      `json:"histories"`
	Permissions    int         `json:"permissions"`
	Banned         bool        `json:"banned"`
	Inactive       bool        `json:"inactive"`
	ScoreStats     ScoreStats  `json:"scoreStats"`
	FirstSeen      string      `json:"firstSeen"`
}

type ScoreStats struct {
	TotalScore            int     `json:"totalScore"`
	TotalRankedScore      int     `json:"totalRankedScore"`
	AverageRankedAccuracy float64 `json:"averageRankedAccuracy"`
	TotalPlayCount        int     `json:"totalPlayCount"`
	RankedPlayCount       int     `json:"rankedPlayCount"`
	ReplaysWatched        int     `json:"replaysWatched"`
}

type PlayerScores struct {
	PlayerScores []PlayerScore `json:"playerScores"`
}

type PlayerScore struct {
	Score       Score       `json:"score"`
	Leaderboard Leaderboard `json:"leaderboard"`
}

type Score struct {
	ID                    int     `json:"id"`
	LeaderboardPlayerInfo *string `json:"leaderboardPlayerInfo"`
	Rank                  int     `json:"rank"`
	BaseScore             int     `json:"baseScore"`
	ModifiedScore         int     `json:"modifiedScore"`
	PP                    float64 `json:"pp"`
	Weight                float64 `json:"weight"`
	Modifiers             string  `json:"modifiers"`
	Multiplier            float64 `json:"multiplier"`
	BadCuts               int     `json:"badCuts"`
	MissedNotes           int     `json:"missedNotes"`
	MaxCombo              int     `json:"maxCombo"`
	FullCombo             bool    `json:"fullCombo"`
	Hmd                   int     `json:"hmd"`
	TimeSet               string  `json:"timeSet"`
	HasReplay             bool    `json:"hasReplay"`
	DeviceHmd             string  `json:"deviceHmd"`
	DeviceControllerLeft  string  `json:"deviceControllerLeft"`
	DeviceControllerRight string  `json:"deviceControllerRight"`
}

type Leaderboard struct {
	ID                int        `json:"id"`
	SongHash          string     `json:"songHash"`
	SongName          string     `json:"songName"`
	SongSubName       string     `json:"songSubName"`
	SongAuthorName    string     `json:"songAuthorName"`
	LevelAuthorName   string     `json:"levelAuthorName"`
	Difficulty        Difficulty `json:"difficulty"`
	MaxScore          int        `json:"maxScore"`
	CreatedDate       string     `json:"createdDate"`
	RankedDate        *string    `json:"rankedDate"`
	QualifiedDate     *string    `json:"qualifiedDate"`
	LovedDate         *string    `json:"lovedDate"`
	Ranked            bool       `json:"ranked"`
	Qualified         bool       `json:"qualified"`
	Loved             bool       `json:"loved"`
	MaxPP             float64    `json:"maxPP"`
	Stars             float64    `json:"stars"`
	Plays             int        `json:"plays"`
	DailyPlays        int        `json:"dailyPlays"`
	PositiveModifiers bool       `json:"positiveModifiers"`
	PlayerScore       *string    `json:"playerScore"`
	CoverImage        string     `json:"coverImage"`
	Difficulties      *string    `json:"difficulties"`
}

type Difficulty struct {
	LeaderboardID int    `json:"leaderboardId"`
	Difficulty    int    `json:"difficulty"`
	GameMode      string `json:"gameMode"`
	DifficultyRaw string `json:"difficultyRaw"`
}

type PlayerResult struct {
	Player         Player
	OriginalRank   int
	TotalPP        float64
	TotalScores    int
	AquafleeScores int
	ValidScores    int
	PPDifference   float64
}

func fetchPlayerScores(playerID string) (float64, int, int, error) {
	url := fmt.Sprintf("https://scoresaber.com/api/player/%s/scores?limit=100", playerID)

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}

	var playerScores PlayerScores
	err = json.Unmarshal(body, &playerScores)
	if err != nil {
		return 0, 0, 0, err
	}

	totalPP := 0.0
	totalScores := len(playerScores.PlayerScores)
	aquafleeScores := 0

	for _, playerScore := range playerScores.PlayerScores {
		levelAuthor := strings.ToLower(playerScore.Leaderboard.LevelAuthorName)
		if strings.Contains(levelAuthor, "aquaflee") {
			aquafleeScores++
			continue
		}

		pp := playerScore.Score.PP
		weight := playerScore.Score.Weight
		weightedPP := pp * weight
		totalPP += weightedPP
	}

	return totalPP, totalScores, aquafleeScores, nil
}

func main() {
	// Fetch top 10 players
	playersURL := "https://scoresaber.com/api/players"

	resp, err := http.Get(playersURL)
	if err != nil {
		log.Fatalf("Error fetching players: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Players API request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading players response: %v", err)
	}

	var playersResponse PlayersResponse
	err = json.Unmarshal(body, &playersResponse)
	if err != nil {
		log.Fatalf("Error parsing players JSON: %v", err)
	}

	file, err := os.Create("top_players.json")
	if err != nil {
		log.Fatalf("Error creating players file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(playersResponse)
	if err != nil {
		log.Fatalf("Error writing players to file: %v", err)
	}

	fmt.Println("\nTop 10 Players - Calculated Total PP from 100 Scores (Excluding Aquaflee Maps):")

	var results []PlayerResult

	for i, player := range playersResponse.Players {
		if i >= 50 {
			break
		}

		fmt.Printf("\nFetching scores for Rank #%d: %s (ID: %s)...\n", player.Rank, player.Name, player.ID)

		totalPP, totalScores, aquafleeScores, err := fetchPlayerScores(player.ID)
		if err != nil {
			fmt.Printf("Error fetching scores for %s: %v\n", player.Name, err)
			continue
		}

		validScores := totalScores - aquafleeScores
		ppDifference := player.PP - totalPP

		result := PlayerResult{
			Player:         player,
			OriginalRank:   player.Rank,
			TotalPP:        totalPP,
			TotalScores:    totalScores,
			AquafleeScores: aquafleeScores,
			ValidScores:    validScores,
			PPDifference:   ppDifference,
		}
		results = append(results, result)

		fmt.Printf("Rank #%d: %s\n", player.Rank, player.Name)
		fmt.Printf("  - Player ID: %s\n", player.ID)
		fmt.Printf("  - Official PP: %.2f\n", player.PP)
		fmt.Printf("  - Total Scores: %d\n", totalScores)
		fmt.Printf("  - Aquaflee Scores Removed: %d\n", aquafleeScores)
		fmt.Printf("  - Valid Scores Used: %d\n", validScores)
		fmt.Printf("  - Calculated Total PP (without Aquaflee): %.4f\n", totalPP)
		fmt.Printf("  - PP Difference: %.4f\n", ppDifference)

		time.Sleep(100 * time.Millisecond)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalPP > results[j].TotalPP
	})

	fmt.Println("NEW TOP 50 RANKING (Based on PP without Aquaflee Maps):")

	for i, result := range results {
		rankChange := result.OriginalRank - (i + 1)
		rankChangeStr := ""
		if rankChange > 0 {
			rankChangeStr = fmt.Sprintf(" (↑%d)", rankChange)
		} else if rankChange < 0 {
			rankChangeStr = fmt.Sprintf(" (↓%d)", -rankChange)
		} else {
			rankChangeStr = " (=)"
		}

		fmt.Printf("\n#%d: %s%s\n", i+1, result.Player.Name, rankChangeStr)
		fmt.Printf("    Original Rank: #%d\n", result.OriginalRank)
		fmt.Printf("    Official PP: %.2f\n", result.Player.PP)
		fmt.Printf("    PP without Aquaflee: %.4f\n", result.TotalPP)
		fmt.Printf("    PP Lost to Aquaflee: %.4f (%.2f%%)\n",
			result.PPDifference,
			(result.PPDifference/result.Player.PP)*100)
		fmt.Printf("    Aquaflee Scores: %d/%d\n", result.AquafleeScores, result.TotalScores)
	}

	fmt.Println("RANKING CHANGES SUMMARY:")

	for i, result := range results {
		newRank := i + 1
		rankChange := result.OriginalRank - newRank

		if rankChange != 0 {
			direction := "down"
			if rankChange > 0 {
				direction = "up"
			}
			fmt.Printf("%s: #%d → #%d (moved %s %d positions)\n",
				result.Player.Name, result.OriginalRank, newRank, direction, abs(rankChange))
		} else {
			fmt.Printf("%s: #%d (no change)\n", result.Player.Name, result.OriginalRank)
		}
	}

	fmt.Println("\nAnalysis complete!")
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
