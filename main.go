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
	"golang.org/x/term"
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
	version   = "1.0.2"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	reset  = "\033[0m"
	showUsage bool
	noColor   bool
	showVersion bool
)

func main() {

	printCopyright()
	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		// Handle cleanup or other tasks before exiting
		fmt.Println("\nTerminating the application...")
		os.Exit(0)
	}()

	for {
		fmt.Println("Select an option:")
		fmt.Println("1. Update the startup of already made servers to the egg startup")
		fmt.Println("2. Stop stuck server transfers")
		fmt.Println("3. Exit")

		var choice int
		fmt.Print("Enter your choice: ")
		fmt.Scan(&choice)
		
		// Consume the newline character
		fmt.Scanln()

		switch choice {
		case 1:
			fmt.Println("Selected: Update the startup of already made servers to the egg startup")
			updateServerStartup()
		case 2:
			fmt.Println("Selected: Stop stuck server transfers")
			stopStuckTransfers()
			// Call your specific function for Option 4 here
		case 3:
			fmt.Println("Exiting the program.")
			os.Exit(0)
		default:
			fmt.Println("Invalid choice. Please select a valid option.")
		}
	}
}


func connectToDatabase() (*sql.DB, error) {
	// Prompt for MySQL connection details
	host := GetUserInput("Enter MySQL host ", "127.0.0.1")
	port := GetUserInput("Enter MySQL port ", "3306")
	user := GetUserInput("Enter MySQL username ", "pterodactyl")

	// Prompt for MySQL password
	password := GetPasswordInput("Enter MySQL password: ")

	// Prompt for Pterodactyl database name
	pterodactylDBName := GetUserInput("Enter Pterodactyl database name ", "panel")

	// Build the MySQL DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, pterodactylDBName)

	// Connect to MySQL
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error connecting to MySQL: %v", err)
	}

	// Check if the connection is successful
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error pinging MySQL server: %v", err)
	}

	if noColor {
		fmt.Println("Connected to MySQL server successfully!")
	} else {
		printFormatted(green, "Connected to MySQL server successfully!\n")
	}

	return db, nil
}


func updateServerStartup(){
		// Establish database connection
		db, err := connectToDatabase()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
	
		// Prompt for egg ID
		eggID := GetUserInput("Enter egg ID: ", "")
	
		// Query the startup field for the specified egg ID from the eggs table
		eggQuery := "SELECT startup FROM eggs WHERE id = ?"
	
		var eggStartup string
		err = db.QueryRow(eggQuery, eggID).Scan(&eggStartup)
		if err != nil {
			if err == sql.ErrNoRows {
				printFormatted(red, "Egg ID not found\n")
				return
			} else {
				log.Fatal("Error retrieving startup value from eggs table:", err)
			}
		}
	
	
		fmt.Printf("\nStartup value for Egg ID %s: %s\n", eggID, eggStartup)
	
		// Query servers with the specified egg ID
		serverQuery := "SELECT uuidShort, name, startup FROM servers WHERE egg_id = ?"
	

		rows, err := db.Query(serverQuery, eggID)
		if err != nil {
			log.Fatal("Error querying servers:", err)
		}
		defer rows.Close()
	
				// Check if there are no servers for the specified egg ID
		if !rows.Next() {
			printFormatted(red, "No servers found for Egg ID %s\n", eggID)
			return
		}
		
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
				fmt.Print("Do you want to update the startup for this server? (y/n/A): ")
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
				} else if updateChoice == "A" || updateChoice == "a" {
					// Update the startup for all subsequent servers
					updateQuery := "UPDATE servers SET startup = ? WHERE egg_id = ? AND startup != ?"
					_, err := db.Exec(updateQuery, eggStartup, eggID, eggStartup)
					if err != nil {
						log.Fatal("Error updating startup for servers:", err)
					}
					printFormatted(green, "Startup updated for all subsequent servers with Egg ID %s\n", eggID)
					break // Exit the loop after updating all servers
				}
			}
		}
		
		// Exit the program after processing rows
		os.Exit(0)
}

func stopStuckTransfers() {
		// Establish database connection
		db, err := connectToDatabase()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
	
		// Query for failed transfers
		transfersQuery := "SELECT id, server_id FROM server_transfers WHERE successful IS NULL"
		rows, err := db.Query(transfersQuery)
		if err != nil {
			log.Fatal("Error querying failed transfers:", err)
		}
		defer rows.Close()
	
		// Check if there are no failed transfers
		if !rows.Next() {
			printFormatted(red, "No stuck transfers found\n")
			return
		}
	
		fmt.Println("Stuck transfers found:\n")
	
		for rows.Next() {
			var transferID, serverID string
			if err := rows.Scan(&transferID, &serverID); err != nil {
				log.Fatal("Error scanning row:", err)
			}
	
			// Get server name from the servers table
			serverNameQuery := "SELECT name FROM servers WHERE id = ?"
			var serverName string
			err := db.QueryRow(serverNameQuery, serverID).Scan(&serverName)
			if err != nil {
				log.Fatal("Error retrieving server name:", err)
			}
	
			// Display information about the stuck transfer
			fmt.Printf("Transfer ID: %s\n", transferID)
			fmt.Printf("Server ID: %s\n", serverID)
			fmt.Printf("Server Name: %s\n", serverName)
	
			// Prompt the user to remove the stuck transfer
			fmt.Print("Do you want to remove this transfer? (y/n): ")
			var removeChoice string
			fmt.Scanln(&removeChoice)
	
			if removeChoice == "y" || removeChoice == "Y" {
				// Remove the stuck transfer entry
				removeTransferQuery := "DELETE FROM server_transfers WHERE id = ?"
				_, err := db.Exec(removeTransferQuery, transferID)
				if err != nil {
					log.Fatal("Error removing transfer:", err)
				}
				printFormatted(green, "Transfer removed\n")
			}

		printFormatted(green, "No more stuck transfers")

	}
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
func printCopyright() {
	
	str := `
	_______  __   __  ___   __    _  _______  _______  __    _  _______  __   __  ______   _______ 
	|       ||  | |  ||   | |  |  | ||       ||       ||  |  | ||       ||  | |  ||      | |  _    |
	|   _   ||  | |  ||   | |   |_| ||_     _||    ___||   |_| ||   _   ||  |_|  ||  _    || | |   |
	|  | |  ||  |_|  ||   | |       |  |   |  |   |___ |       ||  | |  ||       || | |   || | |   |
	|  |_|  ||       ||   | |  _    |  |   |  |    ___||  _    ||  |_|  ||       || |_|   || |_|   |
	|      | |       ||   | | | |   |  |   |  |   |___ | | |   ||      |  |     | |       ||       |
	|____||_||_______||___| |_|  |__|  |___|  |_______||_|  |__||____||_|  |___|  |______| |_______|
																																																					 
	`
	fmt.Println(str)
}

// GetUserInput prompts the user for input with a given prompt and provides a default value.
func GetUserInput(prompt, defaultValue string) string {
	fmt.Print(prompt)

	// Set default value
	if defaultValue != "" {
		fmt.Printf("(default: %s): ", defaultValue)
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
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
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