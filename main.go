package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	// Connects to provided files:
	// - client.go: Provides Client, NewClient, SetLogLatencyFunc
	// - clans.go: Provides ClanService, CurrentWar, Members
	// - locations.go: Provides LocationService, PlayerRankings
	// - players.go: Provides Player, Card, PlayerClan, PlayerService
	// - tournaments.go: Provides TournamentService, TournamentsService, Tournament
	"github.com/fiskie/go-clash/clash"
)

// Tower represents a tower with stats as per TCR Appendix
type Tower struct {
	Type     string  `json:"type"`
	HP       int     `json:"hp"`
	ATK      int     `json:"atk"`
	DEF      int     `json:"def"`
	CRIT     float64 `json:"crit"` // Crit chance (0.1 for King, 0.05 for Guard)
	MaxHP    int     `json:"max_hp"`
}

// CardStats defines detailed card information
type CardStats struct {
	ElixirCost  int
	BaseDamage  int
	HitPoints   int
	CritChance  float64 // Crit chance for the card (0.05 to 0.15)
}

// cardDatabase maps card names to their stats
var cardDatabase = map[string]CardStats{
	"Giant":         {ElixirCost: 5, BaseDamage: 140, HitPoints: 2500, CritChance: 0.05},
	"Musketeer":     {ElixirCost: 4, BaseDamage: 100, HitPoints: 600, CritChance: 0.10},
	"Fireball":      {ElixirCost: 3, BaseDamage: 200, HitPoints: 0, CritChance: 0.15},
	"Archers":       {ElixirCost: 3, BaseDamage: 120, HitPoints: 350, CritChance: 0.08},
	"Knight":        {ElixirCost: 3, BaseDamage: 200, HitPoints: 800, CritChance: 0.08},
	"Arrows":        {ElixirCost: 2, BaseDamage: 100, HitPoints: 0, CritChance: 0.10},
	"Goblin Barrel": {ElixirCost: 3, BaseDamage: 60, HitPoints: 150, CritChance: 0.07},
	"Minions":       {ElixirCost: 3, BaseDamage: 70, HitPoints: 200, CritChance: 0.09},
}

// ReplayData stores simulated replay information
type ReplayData struct {
	Actions []string // List of actions (cards used)
}

// MockPlayer simulates the clash.Player structure from JSON
// Connects to players.go: Mirrors clash.Player for Test Mode
type MockPlayer struct {
	Tag         string           `json:"tag"`
	Name        string           `json:"name"`
	ExpLevel    int              `json:"expLevel"`
	Trophies    int              `json:"trophies"`
	CurrentDeck []clash.Card     `json:"currentDeck"` // From players.go: clash.Card
	Clan        clash.PlayerClan `json:"clan"`        // From players.go: clash.PlayerClan
}

// GameState stores the game state
type GameState struct {
	PlayerTowers []Tower
	EnemyTowers  []Tower
	PlayerElixir float64
	EnemyElixir  float64
}

func main() {
	// Initialize logger
	// Connects to client.go: Used for logging API errors/info
	logger := &Logger{
		infoLog:  log.New(os.Stdout, "INFO: ", log.LstdFlags),
		errorLog: log.New(os.Stderr, "ERROR: ", log.LstdFlags),
	}

	// Declare player
	// Connects to players.go: clash.Player stores player data
	var player clash.Player

	// Select mode (test or live)
	fmt.Print("Select mode (1: Live Mode, 2: Test Mode): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	modeChoice := strings.TrimSpace(scanner.Text())
	isTestMode := modeChoice == "2"

	var client *clash.Client
	var mockPlayers []MockPlayer

	if isTestMode {
		// Read data from player.json
		data, err := ioutil.ReadFile("player.json")
		if err != nil {
			logger.Error("Error reading player.json: %v", err)
			fmt.Println("Unable to read player.json. Exiting program.")
			return
		}
		if err := json.Unmarshal(data, &mockPlayers); err != nil {
			logger.Error("Error parsing player.json: %v", err)
			fmt.Println("Unable to parse player.json. Exiting program.")
			return
		}
	} else {
		// Enter API token
		fmt.Print("Enter your API Token: ")
		scanner.Scan()
		apiToken := strings.TrimSpace(scanner.Text())
		if apiToken == "" {
			logger.Error("API Token cannot be empty")
			return
		}

		// Connects to client.go: Initializes clash.Client with NewClient
		client = clash.NewClient(apiToken, logger.Error, logger.Info)
		// Connects to client.go: Sets API latency logging
		client.SetLogLatencyFunc(func(statusCode, method, host, path string, elapsed time.Duration) {
			logger.Info("Latency %s %s -> %s (%s): %v", method, host, path, statusCode, elapsed)
		})
	}

	// Enter player tag
	for {
		fmt.Print("Enter player tag ( #ABC123): ")
		scanner.Scan()
		playerTag := strings.TrimSpace(scanner.Text())
		if playerTag == "" {
			logger.Error("Player tag cannot be empty")
			fmt.Println("Player tag cannot be empty. Please try again.")
			continue
		}

		// Normalize tag
		playerTag = strings.Replace(playerTag, "#", "%23", -1)

		// Fetch player information
		if isTestMode {
			// Find player in mockPlayers
			for _, mock := range mockPlayers {
				if mock.Tag == strings.Replace(playerTag, "%23", "#", -1) {
					player = clash.Player{
						Tag:         mock.Tag,
						Name:        mock.Name,
						ExpLevel:    mock.ExpLevel,
						Trophies:    mock.Trophies,
						CurrentDeck: mock.CurrentDeck,
						Clan:        mock.Clan,
					}
					break
				}
			}
			if player.Tag == "" {
				logger.Error("Player tag %s not found in player.json", playerTag)
				fmt.Println("Player not found in player.json. Please enter a valid tag (e.g., #PLAYER1 or #PLAYER2).")
				continue
			}
		} else {
			// Connects to players.go: Fetches player data via client.Player(playerTag).Get()
			var err error
			player, err = client.Player(playerTag).Get()
			if err != nil {
				logger.Error("Error fetching player data: %v", err)
				fmt.Println("Player not found. Check tag or API token. Please try again.")
				continue
			}
		}
		break // Tag found, exit loop
	}

	// Welcome player
	fmt.Printf("\nWelcome %s (Level %d, Trophies: %d)!\n", player.Name, player.ExpLevel, player.Trophies)
	fmt.Println("Starting Clash Royale in terminal!")

	// Main loop
	for {
		// Select game mode
		fmt.Println("\nSelect game mode:")
		fmt.Println("1. Normal Mode (Battle with clan members)")
		if !isTestMode {
			fmt.Println("2. Tournament Mode (Battle in tournaments)")
			fmt.Println("3. Ranked Mode (Battle with ranked players)")
			fmt.Println("4. Clan War Mode (Battle in clan wars)")
		}
		// Fixed syntax error: Simplified prompt range
		promptRange := "1"
		if !isTestMode {
			promptRange = "1-4"
		}
		fmt.Printf("Enter number (%s): ", promptRange)
		scanner.Scan()
		mode := strings.TrimSpace(scanner.Text())

		var opponent interface{}
		var opponentName string
		var opponentTrophies int

		if isTestMode {
			// In test mode, only Normal Mode is supported with opponents from player.json
			if mode != "1" {
				fmt.Println("Test Mode only supports Normal Mode. Switching to Normal Mode.")
				mode = "1"
			}
			if len(mockPlayers) > 1 {
				rand.Seed(time.Now().UnixNano())
				for {
					opponent = mockPlayers[rand.Intn(len(mockPlayers))]
					if opponent.(MockPlayer).Tag != player.Tag {
						break
					}
				}
				opponentName = opponent.(MockPlayer).Name
				opponentTrophies = opponent.(MockPlayer).Trophies
			} else {
				opponentName = "Default Enemy"
				opponentTrophies = 1000
			}
		} else {
			switch mode {
			case "1": // Normal Mode
				if player.Clan.Tag != "" {
					// Connects to clans.go: Fetches clan members via client.Clan(player.Clan.Tag).Members()
					members, err := client.Clan(player.Clan.Tag).Members()
					if err == nil && len(members.Items) > 0 {
						rand.Seed(time.Now().UnixNano())
						opponent = members.Items[rand.Intn(len(members.Items))]
						opponentName = opponent.(clash.ClanMember).Name
						opponentTrophies = opponent.(clash.ClanMember).Trophies
					} else {
						fmt.Println("No clan members found. Switching to default opponent.")
					}
				}
			case "2": // Tournament Mode
				fmt.Print("Enter tournament tag (e.g., #XYZ123) or name to search: ")
				scanner.Scan()
				tournamentInput := strings.TrimSpace(scanner.Text())
				if tournamentInput != "" {
					tournamentInput = strings.Replace(tournamentInput, "#", "%23", -1)
					// Connects to tournaments.go: Fetches tournament via client.Tournament(tournamentInput).Get()
					tournament, err := client.Tournament(tournamentInput).Get()
					if err == nil && len(tournament.MembersList) > 0 {
						rand.Seed(time.Now().UnixNano())
						opponent = tournament.MembersList[rand.Intn(len(tournament.MembersList))]
						opponentName = opponent.(clash.TournamentMember).Name
						opponentTrophies = opponent.(clash.TournamentMember).Score
					} else {
						// Connects to tournaments.go: Searches tournaments via client.Tournaments().Search()
						query := &clash.TournamentQuery{Name: tournamentInput}
						tournaments, err := client.Tournaments().Search(query)
						if err == nil && len(tournaments.Items) > 0 {
							tournament = tournaments.Items[0]
							if len(tournament.MembersList) > 0 {
								opponent = tournament.MembersList[rand.Intn(len(tournament.MembersList))]
								opponentName = opponent.(clash.TournamentMember).Name
								opponentTrophies = opponent.(clash.TournamentMember).Score
							}
						} else {
							fmt.Println("Tournament not found. Switching to default opponent.")
						}
					}
				}
			case "3": // Ranked Mode
				fmt.Print("Enter location ID (e.g., global or country code like 57000000): ")
				scanner.Scan()
				locationID := strings.TrimSpace(scanner.Text())
				if locationID == "" {
					locationID = "global"
				}
				// Connects to locations.go: Fetches rankings via client.Location(locationID).PlayerRankings()
				rankings, err := client.Location(locationID).PlayerRankings(&clash.PagedQuery{Limit: 10})
				if err == nil && len(rankings.Items) > 0 {
					rand.Seed(time.Now().UnixNano())
					opponent = rankings.Items[rand.Intn(len(rankings.Items))]
					opponentName = opponent.(clash.PlayerRanking).Name
					opponentTrophies = opponent.(clash.PlayerRanking).Trophies
				} else {
					fmt.Println("No ranked players found. Switching to default opponent.")
				}
			case "4": // Clan War Mode
				if player.Clan.Tag != "" {
					// Connects to clans.go: Fetches clan war via client.Clan(player.Clan.Tag).CurrentWar()
					war, err := client.Clan(player.Clan.Tag).CurrentWar()
					if err == nil && len(war.Participants) > 0 {
						rand.Seed(time.Now().UnixNano())
						opponent = war.Participants[rand.Intn(len(war.Participants))]
						opponentName = opponent.(clash.WarParticipant).Name
						opponentTrophies = 0
					} else {
						fmt.Println("No clan war found. Switching to default opponent.")
					}
				} else {
					fmt.Println("You are not in a clan. Switching to default opponent.")
				}
			default:
				fmt.Println("Invalid mode. Switching to default opponent.")
			}
		}

		// Default opponent
		if opponent == nil {
			opponentName = "Default Enemy"
			opponentTrophies = 1000
		}

		fmt.Printf("Opponent: %s (Trophies: %d)\n", opponentName, opponentTrophies)

		// Play the game and store replay
		// Connects to players.go: Uses clash.Player, clash.Card
		replay := playGame(client, player, opponent, opponentName, logger, isTestMode)

		// Display replay
		fmt.Println("\nMatch replay:")
		for i, action := range replay.Actions {
			fmt.Printf("%d. %s\n", i+1, action)
		}

		fmt.Print("\nContinue playing? (y/n): ")
		scanner.Scan()
		if strings.ToLower(scanner.Text()) != "y" {
			fmt.Println("Thank you for playing!")
			break
		}
	}
}

// playGame implements the game loop
// Connects to players.go: Uses clash.Player, clash.Card
func playGame(client *clash.Client, player clash.Player, opponent interface{}, opponentName string, logger *Logger, isTestMode bool) ReplayData {
	// Display deck
	fmt.Println("\nYour deck:")
	for i, card := range player.CurrentDeck {
		stats, exists := cardDatabase[card.Name]
		if !exists {
			stats = CardStats{ElixirCost: 3, BaseDamage: 50, HitPoints: 100, CritChance: 0.05}
		}
		fmt.Printf("%d. %s (Level %d, Elixir: %d, Damage: %d, HP: %d, Crit: %.0f%%)\n",
			i+1, card.Name, card.Level, stats.ElixirCost, stats.BaseDamage, stats.HitPoints, stats.CritChance*100)
	}

	// Initialize game state with towers
	rand.Seed(time.Now().UnixNano())
	state := GameState{
		PlayerTowers: []Tower{
			{Type: "Guard Tower 1", HP: 1000, ATK: 300, DEF: 100, CRIT: 0.05, MaxHP: 1000},
			{Type: "Guard Tower 2", HP: 1000, ATK: 300, DEF: 100, CRIT: 0.05, MaxHP: 1000},
			{Type: "King Tower", HP: 2000, ATK: 500, DEF: 300, CRIT: 0.1, MaxHP: 2000},
		},
		EnemyTowers: []Tower{
			{Type: "Guard Tower 1", HP: 1000, ATK: 300, DEF: 100, CRIT: 0.05, MaxHP: 1000},
			{Type: "Guard Tower 2", HP: 1000, ATK: 300, DEF: 100, CRIT: 0.05, MaxHP: 1000},
			{Type: "King Tower", HP: 2000, ATK: 500, DEF: 300, CRIT: 0.1, MaxHP: 2000},
		},
		PlayerElixir: 10.0,
		EnemyElixir:  10.0,
	}
	replay := ReplayData{Actions: []string{}}

	// Channels for communication
	inputChan := make(chan string)
	quitChan := make(chan bool)
	// Set elixir regeneration time to 1s
	elixirTick := time.NewTicker(1000 * time.Millisecond) // 1s for 1 elixir
	enemyActionTick := time.NewTicker(5 * time.Second)    // Opponent acts every 5s
	scanner := bufio.NewScanner(os.Stdin)

	// Goroutine to read player input
	go func() {
		for scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			inputChan <- input
			if input == "0" {
				quitChan <- true
				return
			}
		}
	}()

	// Function to clear screen
	clearScreen := func() {
		cmd := exec.Command("clear") // Linux/Mac
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "cls")
		}
		cmd.Stdout = os.Stdout
		cmd.Run()
	}

	// Game loop
	startTime := time.Now()
	for {
		select {
		case <-quitChan:
			fmt.Println("You surrendered!")
			replay.Actions = append(replay.Actions, "Player surrendered")
			elixirTick.Stop()
			enemyActionTick.Stop()
			return replay

		case input := <-inputChan:
			// Parse input
			choice, err := parseInt(input)
			if err != nil || choice < 1 || choice > len(player.CurrentDeck) {
				fmt.Println("Invalid choice. Please select a number from 1 to", len(player.CurrentDeck))
				continue
			}

			// Get selected card
			// Connects to players.go: Uses clash.Card from player.CurrentDeck
			selectedCard := player.CurrentDeck[choice-1]
			stats, exists := cardDatabase[selectedCard.Name]
			if !exists {
				stats = CardStats{ElixirCost: 3, BaseDamage: 50, HitPoints: 100, CritChance: 0.05}
			}

			// Check elixir
			if float64(stats.ElixirCost) > state.PlayerElixir {
				fmt.Printf("Not enough elixir! Need %d, you have %.1f.\n", stats.ElixirCost, state.PlayerElixir)
				continue
			}

			// Calculate and apply damage
			damage, cardCrit, towerCrit := calculateDamage(selectedCard, stats, state.EnemyTowers)
			result := applyDamage(&state.EnemyTowers, damage)
			state.PlayerElixir -= float64(stats.ElixirCost)

			// Save action to replay
			critText := ""
			if cardCrit && towerCrit {
				critText = " (Double CRIT)"
			} else if cardCrit {
				critText = " (Card CRIT)"
			} else if towerCrit {
				critText = " (Tower CRIT)"
			}
			action := fmt.Sprintf("Player used %s (Level %d) dealing %d damage%s to %s", selectedCard.Name, selectedCard.Level, damage, critText, result)
			replay.Actions = append(replay.Actions, action)

			// Display state
			clearScreen()
			displayGameState(state)
			fmt.Printf("You used %s (Level %d) dealing %d damage%s to %s!\n", selectedCard.Name, selectedCard.Level, damage, critText, result)

			// Check for end
			if isKingTowerDestroyed(state.EnemyTowers) {
				fmt.Println("\nCongratulations! You destroyed the opponent's King Tower!")
				replay.Actions = append(replay.Actions, "Player won the match")
				elixirTick.Stop()
				enemyActionTick.Stop()
				return replay
			}

		case <-elixirTick.C:
			// Regenerate elixir
			state.PlayerElixir = minFloat(state.PlayerElixir+1.0, 10.0)
			state.EnemyElixir = minFloat(state.EnemyElixir+1.0, 10.0)

			// Display state
			clearScreen()
			displayGameState(state)
			fmt.Println("Select a card to attack (enter number from 1 to 8, or 0 to surrender): ")

		case <-enemyActionTick.C:
			// Opponent's turn
			if !isKingTowerDestroyed(state.PlayerTowers) {
				var enemyDeck []clash.Card
				if isTestMode && opponent != nil {
					// Connects to players.go: Uses clash.Card from MockPlayer.CurrentDeck
					enemyDeck = opponent.(MockPlayer).CurrentDeck
				} else {
					enemyDeck = player.CurrentDeck // Simulate opponent using same deck
				}
				enemyDamage, cardCrit, towerCrit, enemyAction := simulateEnemyTurn(enemyDeck, state.EnemyElixir, opponentName, state.PlayerTowers)
				if enemyDamage > 0 {
					result := applyDamage(&state.PlayerTowers, enemyDamage)
					state.EnemyElixir -= 3
					critText := ""
					if cardCrit && towerCrit {
						critText = " (Double CRIT)"
					} else if cardCrit {
						critText = " (Card CRIT)"
					} else if towerCrit {
						critText = " (Tower CRIT)"
					}
					fullAction := fmt.Sprintf("%s dealing %d damage%s to %s", enemyAction, enemyDamage, critText, result)
					replay.Actions = append(replay.Actions, fullAction)

					// Display state
					clearScreen()
					displayGameState(state)
					fmt.Printf("Opponent %s used a card dealing %d damage%s to %s!\n", opponentName, enemyDamage, critText, result)
				}

				// Check for end
				if isKingTowerDestroyed(state.PlayerTowers) {
					fmt.Println("\nYou lost! Your King Tower was destroyed.")
					replay.Actions = append(replay.Actions, "Opponent won the match")
					elixirTick.Stop()
					enemyActionTick.Stop()
					return replay
				}
			}
		}

		// Check for draw (3-minute time limit)
		if time.Since(startTime) > 3*time.Minute {
			fmt.Println("\nMatch ended! Draw.")
			replay.Actions = append(replay.Actions, "Match ended in a draw")
			elixirTick.Stop()
			enemyActionTick.Stop()
			return replay
		}
	}
}

// calculateDamage calculates the card's damage with crit chance for both card and tower
func calculateDamage(card clash.Card, stats CardStats, targetTowers []Tower) (int, bool, bool) {
	damage := stats.BaseDamage + (card.Level-1)*10
	randomFactor := rand.Intn(21) - 10

	// Check card crit
	cardCrit := rand.Float64() < stats.CritChance
	towerCrit := false
	critMultiplier := 1.0

	// Check tower crit based on the first available tower
	towerCritChance := 0.05 // Default to Guard Tower crit chance
	for _, tower := range targetTowers {
		if tower.HP > 0 {
			towerCritChance = tower.CRIT
			break
		}
	}
	if rand.Float64() < towerCritChance {
		towerCrit = true
	}

	// Apply crit multipliers
	if cardCrit {
		critMultiplier *= 1.5
	}
	if towerCrit {
		critMultiplier *= 1.2
	}

	totalDamage := int(float64(damage+randomFactor) * critMultiplier)
	return max(1, totalDamage), cardCrit, towerCrit
}

// applyDamage applies damage to the appropriate tower in order: Guard Tower 1, Guard Tower 2, King Tower
func applyDamage(towers *[]Tower, damage int) string {
	// Define the order of towers: Guard Tower 1, Guard Tower 2, King Tower
	targetOrder := []string{"Guard Tower 1", "Guard Tower 2", "King Tower"}
	for _, targetType := range targetOrder {
		for i, tower := range *towers {
			if tower.Type == targetType && tower.HP > 0 {
				(*towers)[i].HP -= damage
				if (*towers)[i].HP < 0 {
					(*towers)[i].HP = 0
				}
				return fmt.Sprintf("%s (HP now %d)", tower.Type, (*towers)[i].HP)
			}
		}
	}
	return "No towers left"
}

// isKingTowerDestroyed checks if the King Tower is destroyed
func isKingTowerDestroyed(towers []Tower) bool {
	for _, tower := range towers {
		if tower.Type == "King Tower" && tower.HP <= 0 {
			return true
		}
	}
	return false
}

// displayGameState prints the current state of towers and elixir
func displayGameState(state GameState) {
	fmt.Println("\n--- Game State ---")
	fmt.Printf("Your Elixir: %.1f | Opponent Elixir: %.1f\n", state.PlayerElixir, state.EnemyElixir)
	fmt.Println("Your Towers:")
	for _, tower := range state.PlayerTowers {
		fmt.Printf("  %s: %d/%d HP\n", tower.Type, max(0, tower.HP), tower.MaxHP)
	}
	fmt.Println("Opponent Towers:")
	for _, tower := range state.EnemyTowers {
		fmt.Printf("  %s: %d/%d HP\n", tower.Type, max(0, tower.HP), tower.MaxHP)
	}
	fmt.Println("-----------------")
}

// simulateEnemyTurn simulates the opponent's turn
// Connects to players.go: Uses clash.Card from enemyDeck
func simulateEnemyTurn(deck []clash.Card, enemyElixir float64, opponentName string, targetTowers []Tower) (int, bool, bool, string) {
	if enemyElixir < 3 || len(deck) == 0 {
		return 0, false, false, fmt.Sprintf("%s skipped turn (not enough elixir)", opponentName)
	}
	card := deck[rand.Intn(len(deck))]
	stats, exists := cardDatabase[card.Name]
	if !exists {
		stats = CardStats{BaseDamage: 50, CritChance: 0.05}
	}
	damage, cardCrit, towerCrit := calculateDamage(card, stats, targetTowers)
	action := fmt.Sprintf("%s used %s (Level %d)", opponentName, card.Name, card.Level)
	return damage, cardCrit, towerCrit, action
}

// parseInt converts string to int
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// minFloat returns the minimum of two float64 values
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Logger provides simple logging functionality
// Connects to client.go: Used for API logging
type Logger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.infoLog != nil {
		l.infoLog.Printf(format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.errorLog != nil {
		l.errorLog.Printf(format, v...)
	}
}