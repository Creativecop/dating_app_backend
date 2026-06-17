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
	flags := flag.NewFlagSet("bootstrap-admin", flag.ExitOnError)
	email := flags.String("email", "", "admin email")
	name := flags.String("name", "", "admin display name")
	if err := flags.Parse(os.Args[1:]); err != nil {
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
	result, err := service.BootstrapSuperAdmin(ctx, admin.BootstrapSuperAdminInput{
		Email:    *email,
		Name:     *name,
		Password: password,
		Secret:   os.Getenv("BOOTSTRAP_ADMIN_SECRET"),
	}, admin.RequestMeta{IPAddress: "bootstrap-cli", UserAgent: "bootstrap-cli"})
	if err != nil {
		log.Fatalf("bootstrap super admin: %v", err)
	}
	fmt.Printf("Bootstrapped SUPER_ADMIN %s (%s).\n", result.Email, result.UUID)
}

func promptPassword() (string, error) {
	if value := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")); value != "" {
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
