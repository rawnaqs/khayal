package internal

import (
	"fmt"
	"os"

	"github.com/rawnaqs/theme"
)

const (
	ExitSuccess = 0
	ExitUser    = 1
	ExitServer  = 2
)

func Successf(format string, args ...any) {
	if len(args) > 0 {
		fmt.Println(theme.SuccessStyle.Render("✓"), fmt.Sprintf(format, args...))
	} else {
		fmt.Println(theme.SuccessStyle.Render("✓"), format)
	}
}

func Errorf(format string, args ...any) {
	if len(args) > 0 {
		fmt.Println(theme.ErrorStyle.Render("✗"), fmt.Sprintf(format, args...))
	} else {
		fmt.Println(theme.ErrorStyle.Render("✗"), format)
	}
}

func Warnf(format string, args ...any) {
	if len(args) > 0 {
		fmt.Println(theme.ProcessingStyle.Render("⚠"), fmt.Sprintf(format, args...))
	} else {
		fmt.Println(theme.ProcessingStyle.Render("⚠"), format)
	}
}

func ActionableError(title string, hints []string) {
	fmt.Println(theme.ErrorStyle.Render("✗"), title)
	for _, hint := range hints {
		fmt.Println("  ", theme.Muted.Render("→"), hint)
	}
}

func ServerUnreachable(host string) {
	ActionableError("cannot reach khayal at "+host, []string{
		"is khayal running?    ssh mac-air khayal start",
		"wrong address?        kl config set host <address>",
		"check status          kl status",
	})
}

func AuthFailed() {
	ActionableError("unauthorized · invalid token", []string{
		"get token from        ~/.config/khayal/config.yaml on server",
		"update token          kl config set token <token>",
	})
}

func Fatal(exitCode int, format string, args ...any) {
	if len(args) > 0 {
		fmt.Println(theme.ErrorStyle.Render("✗"), fmt.Sprintf(format, args...))
	} else {
		fmt.Println(theme.ErrorStyle.Render("✗"), format)
	}
	os.Exit(exitCode)
}

func printError(message, hint string) {
	fmt.Println(theme.ErrorStyle.Render("✗") + " " + theme.Primary.Render(message))
	fmt.Println(theme.Dim.Render("  → " + hint))
}
