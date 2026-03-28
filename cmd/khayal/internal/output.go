package cli

import (
	"fmt"
	"os"

	"github.com/rawnaqs/theme"
)

const (
	ExitSuccess = 0
	ExitUser    = 1
	ExitServer  = 2
	ExitVault   = 3
	ExitDep     = 4
)

func Successf(format string, args ...any) {
	fmt.Println(theme.SuccessStyle.Render("✓"), fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) {
	fmt.Println(theme.ErrorStyle.Render("✗"), fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...any) {
	fmt.Println(theme.ProcessingStyle.Render("⚠"), fmt.Sprintf(format, args...))
}

func Infof(format string, args ...any) {
	fmt.Println(theme.Primary.Render("→"), fmt.Sprintf(format, args...))
}

func Fatal(exitCode int, format string, args ...any) {
	Errorf(format, args...)
	os.Exit(exitCode)
}

func PrintAction(label, value string) {
	fmt.Printf("  %s %s\n", theme.SuccessStyle.Render("✓"), theme.Muted.Render(label))
	if value != "" {
		fmt.Printf("      %s\n", theme.Primary.Render(value))
	}
}

func PrintActionError(label, action string) {
	fmt.Printf("  %s %s\n", theme.ErrorStyle.Render("✗"), theme.Muted.Render(label))
	fmt.Printf("      %s %s\n", theme.Muted.Render("→"), theme.Dim.Render(action))
}

func PrintSection(title string) {
	fmt.Println()
	fmt.Println(theme.Bold.Render(title))
}
