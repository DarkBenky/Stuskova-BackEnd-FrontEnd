package main

import (
    "io"
    "os"
    "strconv"
    "strings"
    "time"

    "net/http"

    "github.com/chzyer/readline"
    "github.com/fatih/color"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func startServer(e *echo.Echo) {
    go func() {
        e.Logger.Fatal(e.Start(":8050"))
    }()
}

func printHelp() {
    help := color.New(color.FgCyan)
    help.Println("Available commands:")
    help.Println("  question <text>          - Set new question")
    help.Println("  time <seconds|last|pause>- Set time left or control timer")
    help.Println("  type <type>              - Set type (pomoc/rozstrel/waiting/end)")
    help.Println("  status                   - Show current question status")
    help.Println("  logging <on/off>         - Enable/disable request logging")
    help.Println("  help                     - Show this help")
    help.Println("  exit                     - Exit the program")
}

type Question struct {
    Question  string        `json:"question"`
    TimeLeft  time.Duration `json:"time_left"`
    Type      string        `json:"type"`
    StartTime time.Time     `json:"start_time"`
}

var Pause = false

var question = Question{
    Question:  "Default question",
    TimeLeft:  time.Second * 30,
    Type:      "pomoc",
    StartTime: time.Now(),
}

func getQuestion(c echo.Context) error {
    q := Question{
        Question:  question.Question,
        TimeLeft:  question.TimeLeft,
        Type:      question.Type,
        StartTime: question.StartTime,
    }
    if Pause {
        return c.JSON(http.StatusOK, q)
    }
    q.TimeLeft = question.TimeLeft - time.Since(question.StartTime)
    if q.TimeLeft < 0 {
        q.TimeLeft = 0
        q.Type = "end"
    }
    return c.JSON(http.StatusOK, q)
}

var loggingEnabled = false

func main() {
    e := echo.New()

    // CORS middleware configuration
    e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins:     []string{"*"}, // Allow all origins
        AllowMethods:     []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
        AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
        AllowCredentials: true,
        MaxAge:           300,
    }))

    // Custom logger middleware that can be disabled
    e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
        Skipper: func(c echo.Context) bool {
            return !loggingEnabled
        },
    }))
    e.Use(middleware.Recover())
    e.GET("/get-question", getQuestion)

    question.TimeLeft = time.Second * 30
    question.Type = "pomoc"
    question.StartTime = time.Now()

    // Start server in goroutine
    startServer(e)

    success := color.New(color.FgGreen)
    errorC := color.New(color.FgRed)
    info := color.New(color.FgYellow)

    // Set up readline for better input experience with multi-command autocomplete
    completer := readline.NewPrefixCompleter(
        readline.PcItem("question"),
        readline.PcItem("time",
            readline.PcItem("last"),
            readline.PcItem("pause"),
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

    // Custom completer to handle multiline commands
    rl, err := readline.NewEx(&readline.Config{
        Prompt:          "\033[32m> \033[0m",
        AutoComplete:    &MultiCommandCompleter{completer},
        HistoryFile:     "/tmp/readline.tmp",
        InterruptPrompt: "^C",
        EOFPrompt:       "exit",
    })
    if err != nil {
        errorC.Printf("Error initializing readline: %v\n", err)
        os.Exit(1)
    }
    defer rl.Close()

    info.Println("Server started. Type 'help' for available commands")

    var lastTime int = 0

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

        // Split input by semicolons to handle multiple commands
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
                question.Question = strings.Join(args[1:], " ")
                question.StartTime = time.Now()
                success.Printf("Question set to: %s\n", question.Question)

            case "time":
                if len(args) != 2 {
                    errorC.Println("Usage: time <seconds|last|pause>")
                    continue
                }
                switch args[1] {
                case "last":
                    question.TimeLeft = time.Duration(lastTime) * time.Second
                    question.StartTime = time.Now()
                    success.Printf("Time left set to: %d seconds\n", lastTime)
                case "pause":
                    Pause = !Pause
                    if Pause {
                        success.Println("Question paused")
                    } else {
                        success.Println("Question unpaused")
                        question.StartTime = time.Now()
                        question.TimeLeft = time.Duration(lastTime) * time.Second
                    }
                default:
                    timeLeft, err := strconv.Atoi(args[1])
                    if err != nil || timeLeft < 0 {
                        errorC.Println("Time must be a positive number")
                        continue
                    }
                    lastTime = timeLeft
                    question.TimeLeft = time.Duration(timeLeft) * time.Second
                    question.StartTime = time.Now()
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
                question.Type = args[1]
                success.Printf("Type set to: %s\n", args[1])

            case "status":
                info.Println("Current question status:")
                info.Printf("Question: %s\n", question.Question)
                timeLeft := question.TimeLeft - time.Since(question.StartTime)
                if timeLeft < 0 {
                    timeLeft = 0
                }
                info.Printf("Time left: %d seconds\n", int(timeLeft.Seconds()))
                info.Printf("Type: %s\n", question.Type)
                info.Printf("Logging: %v\n", loggingEnabled)

            case "help":
                printHelp()

            default:
                errorC.Printf("Unknown command: %s\n", command)
                errorC.Println("Type 'help' for available commands")
            }
        }
    }
}

// Custom AutoCompleter to handle multi-command autocompletion
type MultiCommandCompleter struct {
    root readline.PrefixCompleterInterface
}

func (c *MultiCommandCompleter) Do(line []rune, pos int) ([][]rune, int) {
    // Get the line up to the cursor position
    l := string(line[:pos])

    // Split the line by semicolons
    cmds := strings.Split(l, ";")

    // Get the last command
    lastCmd := cmds[len(cmds)-1]
    lastCmd = strings.TrimSpace(lastCmd)
    lastCmdRunes := []rune(lastCmd)

    // Get the position in the last command
    lastPos := len(lastCmdRunes)

    // Use the root completer for the last command
    return c.root.Do(lastCmdRunes, lastPos)
}