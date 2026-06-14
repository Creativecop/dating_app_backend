package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/neoscoder/aura-backend/internal/admin"
	"github.com/neoscoder/aura-backend/internal/config"
	"github.com/neoscoder/aura-backend/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "create-super-admin":
		createSuperAdmin(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func createSuperAdmin(args []string) {
	flags := flag.NewFlagSet("create-super-admin", flag.ExitOnError)
	email := flags.String("email", "", "admin email")
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}
	if strings.TrimSpace(*email) == "" {
		log.Fatal("--email is required")
	}

	password, err := promptPassword()
	if err != nil {
		log.Fatalf("read password: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("close postgres: %v", err)
		}
	}()

	service := admin.NewService(db, cfg.AdminJWT)
	if err := service.CreateSuperAdmin(ctx, *email, password); err != nil {
		log.Fatalf("create super admin: %v", err)
	}
	fmt.Println("Super admin created successfully.")
}

func promptPassword() (string, error) {
	if value := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")); value != "" {
		return value, nil
	}
	fmt.Print("Password: ")
	first, err := readPassword()
	if err != nil {
		return "", err
	}
	fmt.Print("Confirm password: ")
	second, err := readPassword()
	if err != nil {
		return "", err
	}
	if first != second {
		return "", fmt.Errorf("passwords do not match")
	}
	return first, nil
}

func readPassword() (string, error) {
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		bytes, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(value, "\r\n"), nil
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./cmd/admin create-super-admin --email admin@example.com")
}
