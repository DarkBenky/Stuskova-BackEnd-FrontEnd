package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	flaskServerURL = "http://localhost:5000"
	serverPort     = ":8050"
	historyFile    = "/tmp/readline.tmp"
)

// Question represents the question data structure.
type Question struct {
	Question  string        `json:"question"`
	TimeLeft  time.Duration `json:"time_left"`
	Type      string        `json:"type"`
	StartTime time.Time     `json:"start_time"`
	CountUp   bool          `json:"count_up"`
}

var (
	question       = Question{}
	questionMutex  sync.RWMutex
	pause          = false
	loggingEnabled = false
)

func main() {
	// Initialize the question with default values.
	initializeQuestion()

	// Start the HTTP server.
	e := setupServer()
	startServer(e)

	// Start the command-line interface.
	startCLI()

	// Wait for OS signals to gracefully shut down.
	waitForShutdown(e)
}

func initializeQuestion() {
	questionMutex.Lock()
	defer questionMutex.Unlock()
	question = Question{
		Question:  "Default question",
		TimeLeft:  time.Second * 30,
		Type:      "pomoc",
		StartTime: time.Now(),
		CountUp:   false,
	}
}

func setupServer() *echo.Echo {
	e := echo.New()

	// Configure middleware.
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
	}))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return !loggingEnabled
		},
	}))
	e.Use(middleware.Recover())

	// Define endpoints.
	e.GET("/get-question", getQuestion)
	e.POST("/set-question", setQuestion)

	return e
}

func startServer(e *echo.Echo) {
	go func() {
		if err := e.Start(serverPort); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatalf("Error starting server: %v", err)
		}
	}()
}

func getQuestion(c echo.Context) error {
	questionMutex.RLock()
	defer questionMutex.RUnlock()

	q := question

	if pause {
		return c.JSON(http.StatusOK, q)
	}

	if q.CountUp {
		q.TimeLeft = time.Since(q.StartTime)
	} else {
		q.TimeLeft = q.TimeLeft - time.Since(q.StartTime)
		if q.TimeLeft < 0 {
			q.TimeLeft = 0
			q.Type = "end"
		}
	}

	return c.JSON(http.StatusOK, q)
}

func setQuestion(c echo.Context) error {
	newQuestion := new(Question)
	if err := c.Bind(newQuestion); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := validateQuestion(*newQuestion); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	questionMutex.Lock()
	question = *newQuestion
	question.StartTime = time.Now()
	if question.Type == "end" {
		question.Question = "END"
	}
	questionMutex.Unlock()

	// Send the current question to the Flask server.
	go sendCurrentQuestion()

	return c.JSON(http.StatusOK, question)
}

func validateQuestion(q Question) error {
	if q.TimeLeft < 0 {
		return fmt.Errorf("time_left must be non-negative")
	}
	validTypes := map[string]bool{
		"pomoc":    true,
		"rozstrel": true,
		"waiting":  true,
		"end":      true,
	}
	if !validTypes[q.Type] {
		return fmt.Errorf("invalid type. Must be one of: pomoc, rozstrel, waiting, end")
	}
	return nil
}

func sendCurrentQuestion() {
	questionMutex.RLock()
	jsonData, err := json.Marshal(question)
	questionMutex.RUnlock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}

	resp, err := http.Post(flaskServerURL+"/set-current-question", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending POST request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Failed to send question, status code: %d\n", resp.StatusCode)
	}
}

func startCLI() {
	success := color.New(color.FgGreen)
	errorC := color.New(color.FgRed)
	info := color.New(color.FgYellow)

	completer := readline.NewPrefixCompleter(
		readline.PcItem("question"),
		readline.PcItem("time",
			readline.PcItem("last"),
			readline.PcItem("pause"),
			readline.PcItem("countUp"),
		),
		readline.PcItem("type",
			readline.PcItem("pomoc"),
			readline.PcItem("rozstrel"),
			readline.PcItem("waiting"),
			readline.PcItem("end"),
		),
		readline.PcItem("status"),
		readline.PcItem("logging",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
		readline.PcItem("help"),
		readline.PcItem("exit"),
	)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[32m> \033[0m",
		AutoComplete:    &MultiCommandCompleter{completer},
		HistoryFile:     historyFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		errorC.Printf("Error initializing readline: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	info.Println("Server started. Type 'help' for available commands.")

	var lastTime int

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					break
				} else {
					continue
				}
			} else if err == io.EOF {
				break
			} else {
				errorC.Printf("Error reading input: %v\n", err)
				continue
			}
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Handle multiple commands separated by semicolons.
		commands := strings.Split(input, ";")
		for _, cmd := range commands {
			cmd = strings.TrimSpace(cmd)
			if cmd == "" {
				continue
			}
			args := strings.Fields(cmd)
			command := args[0]
			switch command {
			case "logging":
				if len(args) != 2 {
					errorC.Println("Usage: logging <on/off>")
					continue
				}
				switch args[1] {
				case "on":
					loggingEnabled = true
					success.Println("Request logging enabled")
				case "off":
					loggingEnabled = false
					success.Println("Request logging disabled")
				default:
					errorC.Println("Invalid option. Use 'on' or 'off'")
				}
			case "exit":
				success.Println("Shutting down server...")
				os.Exit(0)
			case "question":
				if len(args) < 2 {
					errorC.Println("Usage: question <text>")
					continue
				}
				questionMutex.Lock()
				question.Question = strings.Join(args[1:], " ")
				question.StartTime = time.Now()
				questionMutex.Unlock()
				success.Printf("Question set to: %s\n", question.Question)

				// Send the current question to the Flask server.
				go sendCurrentQuestion()
			case "time":
				if len(args) != 2 {
					errorC.Println("Usage: time <seconds|last|pause|countUp>")
					continue
				}
				switch args[1] {
				case "last":
					questionMutex.Lock()
					question.TimeLeft = time.Duration(lastTime) * time.Second
					question.StartTime = time.Now()
					question.CountUp = false
					questionMutex.Unlock()
					success.Printf("Time left set to: %d seconds\n", lastTime)
				case "pause":
					pause = !pause
					if pause {
						success.Println("Question paused")
					} else {
						success.Println("Question unpaused")
						questionMutex.Lock()
						question.StartTime = time.Now()
						questionMutex.Unlock()
					}
				case "countUp":
					questionMutex.Lock()
					question.StartTime = time.Now()
					question.CountUp = true
					questionMutex.Unlock()
					success.Println("Counting up")
				default:
					timeLeft, err := strconv.Atoi(args[1])
					if err != nil || timeLeft < 0 {
						errorC.Println("Time must be a non-negative integer")
						continue
					}
					lastTime = timeLeft
					questionMutex.Lock()
					question.TimeLeft = time.Duration(timeLeft) * time.Second
					question.StartTime = time.Now()
					question.CountUp = false
					questionMutex.Unlock()
					success.Printf("Time left set to: %d seconds\n", timeLeft)
				}
			case "type":
				if len(args) != 2 {
					errorC.Println("Usage: type <pomoc/rozstrel/waiting/end>")
					continue
				}
				validTypes := map[string]bool{
					"pomoc":    true,
					"rozstrel": true,
					"waiting":  true,
					"end":      true,
				}
				if !validTypes[args[1]] {
					errorC.Println("Invalid type. Must be: pomoc, rozstrel, waiting, or end")
					continue
				}
				questionMutex.Lock()
				question.Type = args[1]
				if question.Type == "end" {
					question.Question = "END"
				}
				questionMutex.Unlock()
				success.Printf("Type set to: %s\n", args[1])
			case "status":
				questionMutex.RLock()
				info.Println("Current question status:")
				info.Printf("Question: %s\n", question.Question)
				if question.CountUp {
					elapsedTime := time.Since(question.StartTime)
					info.Printf("Elapsed time: %d seconds\n", int(elapsedTime.Seconds()))
				} else {
					timeLeft := question.TimeLeft - time.Since(question.StartTime)
					if timeLeft < 0 {
						timeLeft = 0
					}
					info.Printf("Time left: %d seconds\n", int(timeLeft.Seconds()))
				}
				info.Printf("Type: %s\n", question.Type)
				info.Printf("Logging: %v\n", loggingEnabled)
				questionMutex.RUnlock()
			case "help":
				printHelp()
			default:
				errorC.Printf("Unknown command: %s\n", command)
				errorC.Println("Type 'help' for available commands")
			}
		}
	}
}

func waitForShutdown(e *echo.Echo) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal("Server Shutdown Failed:", err)
	}
}

func printHelp() {
	help := color.New(color.FgCyan)
	help.Println("Available commands:")
	help.Println("  question <text>          - Set new question")
	help.Println("  time <seconds|last|pause|countUp> - Set time left or control timer")
	help.Println("  type <type>              - Set type (pomoc/rozstrel/waiting/end)")
	help.Println("  status                   - Show current question status")
	help.Println("  logging <on/off>         - Enable/disable request logging")
	help.Println("  help                     - Show this help")
	help.Println("  exit                     - Exit the program")
}

// MultiCommandCompleter handles autocomplete for multiple commands.
type MultiCommandCompleter struct {
	root readline.PrefixCompleterInterface
}

func (c *MultiCommandCompleter) Do(line []rune, pos int) ([][]rune, int) {
	l := string(line[:pos])
	cmds := strings.Split(l, ";")
	lastCmd := cmds[len(cmds)-1]
	lastCmd = strings.TrimSpace(lastCmd)
	lastCmdRunes := []rune(lastCmd)
	lastPos := len(lastCmdRunes)
	return c.root.Do(lastCmdRunes, lastPos)
}
