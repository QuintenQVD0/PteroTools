package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pmezard/go-difflib/difflib"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/common-nighthawk/go-figure"
)

func init() {
	flag.BoolVar(&showUsage, "h", false, "Show usage information")
	flag.BoolVar(&noColor, "no-color", false, "Disable colorized output")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	if showVersion {
		printVersion()
		os.Exit(0)
	}

	if showUsage {
		flag.Usage()
		os.Exit(0)
	}
}

var (
	version   = "1.0.0"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	reset  = "\033[0m"
	showUsage bool
	noColor   bool
	showVersion bool
)

func main() {

	printCopyright("QuintenQVD0")
	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		// Handle cleanup or other tasks before exiting
		fmt.Println("\nTerminating the application...")
		os.Exit(0)
	}()

	// Prompt for MySQL connection details
	host := GetUserInput("Enter MySQL host: ", "10.0.0.36")
	port := GetUserInput("Enter MySQL port: ", "3306")
	user := GetUserInput("Enter MySQL username: ", "pterodactyl2")

	// Prompt for MySQL password
	password := GetPasswordInput("Enter MySQL password: ")

	// Prompt for Pterodactyl database name
	pterodactylDBName := GetUserInput("Enter Pterodactyl database name (default: panel): ", "panel")

	// Build the MySQL DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, pterodactylDBName)

	// Connect to MySQL
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error connecting to MySQL:", err)
	}
	defer db.Close()

	// Check if the connection is successful
	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging MySQL server:", err)
	}

	if noColor {
		fmt.Println("Connected to MySQL server successfully!")
	} else {
		printFormatted(green, "Connected to MySQL server successfully!\n")
	}

	// Prompt for egg ID
	eggID := GetUserInput("Enter egg ID: ", "")

	// Query the startup field for the specified egg ID from the eggs table
	eggQuery := "SELECT startup FROM eggs WHERE id = ?"

	var eggStartup string
	err = db.QueryRow(eggQuery, eggID).Scan(&eggStartup)
	if err != nil {
		log.Fatal("Error retrieving startup value from eggs table:", err)
	}

	fmt.Printf("\nStartup value for Egg ID %s: %s\n", eggID, eggStartup)

	// Query servers with the specified egg ID
	serverQuery := "SELECT uuidShort, name, startup FROM servers WHERE egg_id = ?"

	rows, err := db.Query(serverQuery, eggID)
	if err != nil {
		log.Fatal("Error querying servers:", err)
	}
	defer rows.Close()

	fmt.Printf("Servers using Egg ID %s:\n\n", eggID)

	for rows.Next() {
		var uuidShort, name, serverStartup string
		if err := rows.Scan(&uuidShort, &name, &serverStartup); err != nil {
			log.Fatal("Error scanning row:", err)
		}
	
		// Compare startup value from eggs table with startup value from servers table
		if eggStartup == serverStartup {
			printFormatted(green, "UUID Short: %s, Name: %s, Startup Matches\n", uuidShort, name)
		} else {
			printFormatted(red, "UUID Short: %s, Name: %s, Startup Mismatch\n", uuidShort, name)
	
			// Show the difference between eggStartup and serverStartup
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(eggStartup),
				B:        difflib.SplitLines(serverStartup),
				FromFile: "Egg Startup",
				ToFile:   "Server Startup",
				Context:  3,
			}
	
			diffText, _ := difflib.GetUnifiedDiffString(diff)
			printFormatted(yellow, "Difference:\n%s", diffText)
	
			// Prompt the user to update the startup
			fmt.Print("Do you want to update the startup for this server? (y/n): ")
			var updateChoice string
			fmt.Scanln(&updateChoice)
	
			if updateChoice == "y" || updateChoice == "Y" {
				// Update the startup for the server with the latest one from the egg
				updateQuery := "UPDATE servers SET startup = ? WHERE uuidShort = ?"
				_, err := db.Exec(updateQuery, eggStartup, uuidShort)
				if err != nil {
					log.Fatal("Error updating startup for server:", err)
				}
				printFormatted(green, "Startup updated for server with UUID Short %s\n", uuidShort)
			}
		}
	}
	// Exit the program after processing rows
	os.Exit(0)
	// Wait for user input or signal to exit
	select {}
}

// printFormatted prints a formatted string with color (if enabled)
func printFormatted(colorCode, format string, a ...interface{}) {
	if noColor {
		fmt.Printf(format, a...)
	} else {
		fmt.Printf(colorCode+format+reset, a...)
	}
}

// Print Copyright
func printCopyright(text string) {
	figure.NewFigure(text, "", true).Print()
	fmt.Println()
}

// GetUserInput prompts the user for input with a given prompt and provides a default value.
func GetUserInput(prompt, defaultValue string) string {
	fmt.Print(prompt)

	// Set default value
	if defaultValue != "" {
		fmt.Printf("(default: %s) ", defaultValue)
	}

	// Read user input
	var input string
	fmt.Scanln(&input)

	// Use default value if input is empty
	if input == "" {
		return defaultValue
	}

	return input
}

// GetPasswordInput prompts the user for a password without showing it on the screen.
func GetPasswordInput(prompt string) string {
	fmt.Print(prompt)

	// Read password without showing it
	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal("Error reading password:", err)
	}

	// Print a new line after reading password
	fmt.Println()

	return string(password)
}

func printVersion() {
	fmt.Printf("Version %s\n", version)
}