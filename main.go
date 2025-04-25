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

	"github.com/fiskie/go-clash/clash"
)

// CardStats defines detailed card information
type CardStats struct {
	ElixirCost int
	BaseDamage int
	HitPoints  int
}

var cardDatabase = map[string]CardStats{
	"Giant":         {ElixirCost: 6, BaseDamage: 140, HitPoints: 2500},
	"Musketeer":     {ElixirCost: 4, BaseDamage: 100, HitPoints: 600},
	"Fireball":      {ElixirCost: 3, BaseDamage: 200, HitPoints: 0},
	"Archers":       {ElixirCost: 3, BaseDamage: 120, HitPoints: 350},
	"Knight":        {ElixirCost: 3, BaseDamage: 200, HitPoints: 800},
	"Arrows":        {ElixirCost: 2, BaseDamage: 100, HitPoints: 0},
	"Goblin Barrel": {ElixirCost: 3, BaseDamage: 60, HitPoints: 150},
	"Minions":       {ElixirCost: 3, BaseDamage: 70, HitPoints: 200},
}

// ReplayData stores simulated replay information
type ReplayData struct {
	Actions []string // List of actions (cards used)
}

// MockPlayer simulates the clash.Player structure from JSON
type MockPlayer struct {
	Tag         string           `json:"tag"`
	Name        string           `json:"name"`
	ExpLevel    int              `json:"expLevel"`
	Trophies    int              `json:"trophies"`
	CurrentDeck []clash.Card     `json:"currentDeck"`
	Clan        clash.PlayerClan `json:"clan"`
}

// GameState stores the game state
type GameState struct {
	PlayerTowerHP int
	EnemyTowerHP  int
	PlayerElixir  float64
	EnemyElixir   float64
}

func main() {
	// Initialize logger
	logger := &Logger{
		infoLog:  log.New(os.Stdout, "INFO: ", log.LstdFlags),
		errorLog: log.New(os.Stderr, "ERROR: ", log.LstdFlags),
	}

	// Declare player at the function scope
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

		// Create client
		client = clash.NewClient(apiToken, logger.Error, logger.Info)
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
		fmt.Print("Enter number (1" + func() string {
			if isTestMode {
				return ")"
			}
			return "-4)"
		}() + ": ")
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
					tournament, err := client.Tournament(tournamentInput).Get()
					if err == nil && len(tournament.MembersList) > 0 {
						rand.Seed(time.Now().UnixNano())
						opponent = tournament.MembersList[rand.Intn(len(tournament.MembersList))]
						opponentName = opponent.(clash.TournamentMember).Name
						opponentTrophies = opponent.(clash.TournamentMember).Score
					} else {
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

func playGame(client *clash.Client, player clash.Player, opponent interface{}, opponentName string, logger *Logger, isTestMode bool) ReplayData {
	// Display deck
	fmt.Println("\nYour deck:")
	for i, card := range player.CurrentDeck {
		stats, exists := cardDatabase[card.Name]
		if !exists {
			stats = CardStats{ElixirCost: 3, BaseDamage: 50, HitPoints: 100}
		}
		fmt.Printf("%d. %s (Level %d, Elixir: %d, Damage: %d, HP: %d)\n",
			i+1, card.Name, card.Level, stats.ElixirCost, stats.BaseDamage, stats.HitPoints)
	}

	// Initialize game state
	rand.Seed(time.Now().UnixNano())
	state := GameState{
		PlayerTowerHP: 2000,
		EnemyTowerHP:  2000,
		PlayerElixir:  10.0,
		EnemyElixir:   10.0,
	}
	replay := ReplayData{Actions: []string{}}

	// Channels for communication
	inputChan := make(chan string)
	quitChan := make(chan bool)
	// Set elixir regeneration time to 2s
	elixirTick := time.NewTicker(2000 * time.Millisecond) // 2s for 1 elixir
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
			choice, err := parseInt(input)
			if err != nil || choice < 1 || choice > len(player.CurrentDeck) {
				fmt.Println("Invalid choice. Please select a number from 1 to", len(player.CurrentDeck))
				continue
			}

			// Get selected card
			selectedCard := player.CurrentDeck[choice-1]
			stats, exists := cardDatabase[selectedCard.Name]
			if !exists {
				stats = CardStats{ElixirCost: 3, BaseDamage: 50, HitPoints: 100}
			}

			// Check elixir
			if float64(stats.ElixirCost) > state.PlayerElixir {
				fmt.Printf("Not enough elixir! Need %d, you have %.1f.\n", stats.ElixirCost, state.PlayerElixir)
				continue
			}

			// Calculate damage
			damage := calculateDamage(selectedCard, stats)
			state.EnemyTowerHP -= damage
			state.PlayerElixir -= float64(stats.ElixirCost)

			// Save action to replay
			action := fmt.Sprintf("Player used %s (Level %d) dealing %d damage", selectedCard.Name, selectedCard.Level, damage)
			replay.Actions = append(replay.Actions, action)

			// Display state
			clearScreen()
			fmt.Printf("\nYour Tower: %d HP | Opponent Tower: %d HP\n", max(0, state.PlayerTowerHP), max(0, state.EnemyTowerHP))
			fmt.Printf("Your Elixir: %.1f | Opponent Elixir: %.1f\n", state.PlayerElixir, state.EnemyElixir)
			fmt.Printf("You used %s (Level %d) dealing %d damage!\n", selectedCard.Name, selectedCard.Level, damage)

			// Check for end
			if state.EnemyTowerHP <= 0 {
				fmt.Println("\nCongratulations! You destroyed the opponent's tower!")
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
			fmt.Printf("\nYour Tower: %d HP | Opponent Tower: %d HP\n", max(0, state.PlayerTowerHP), max(0, state.EnemyTowerHP))
			fmt.Printf("Your Elixir: %.1f | Opponent Elixir: %.1f\n", state.PlayerElixir, state.EnemyElixir)
			fmt.Println("Select a card to attack (enter number from 1 to 8, or 0 to surrender): ")

		case <-enemyActionTick.C:
			// Opponent's turn
			if state.EnemyTowerHP > 0 {
				var enemyDeck []clash.Card
				if isTestMode && opponent != nil {
					enemyDeck = opponent.(MockPlayer).CurrentDeck
				} else {
					enemyDeck = player.CurrentDeck // Simulate opponent using same deck
				}
				enemyDamage, enemyAction := simulateEnemyTurn(enemyDeck, state.EnemyElixir, opponentName)
				if enemyDamage > 0 {
					state.PlayerTowerHP -= enemyDamage
					state.EnemyElixir -= 3
					replay.Actions = append(replay.Actions, enemyAction)

					// Display state
					clearScreen()
					fmt.Printf("\nYour Tower: %d HP | Opponent Tower: %d HP\n", max(0, state.PlayerTowerHP), max(0, state.EnemyTowerHP))
					fmt.Printf("Your Elixir: %.1f | Opponent Elixir: %.1f\n", state.PlayerElixir, state.EnemyElixir)
					fmt.Printf("Opponent %s used a card dealing %d damage!\n", opponentName, enemyDamage)
				}

				// Check for end
				if state.PlayerTowerHP <= 0 {
					fmt.Println("\nYou lost! Your tower was destroyed.")
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

// calculateDamage calculates the card's damage
func calculateDamage(card clash.Card, stats CardStats) int {
	damage := stats.BaseDamage + (card.Level-1)*10
	randomFactor := rand.Intn(21) - 10
	return max(1, damage+randomFactor)
}

// simulateEnemyTurn simulates the opponent's turn
func simulateEnemyTurn(deck []clash.Card, enemyElixir float64, opponentName string) (int, string) {
	if enemyElixir < 3 || len(deck) == 0 {
		return 0, fmt.Sprintf("%s skipped turn (not enough elixir)", opponentName)
	}
	card := deck[rand.Intn(len(deck))]
	stats, exists := cardDatabase[card.Name]
	if !exists {
		stats = CardStats{BaseDamage: 50}
	}
	damage := calculateDamage(card, stats)
	action := fmt.Sprintf("%s used %s (Level %d) dealing %d damage", opponentName, card.Name, card.Level, damage)
	return damage, action
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