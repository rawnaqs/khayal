package cli

import (
	"fmt"

	"github.com/rawnaqs/theme"
)

func ActionableError(title, action1, action2, action3 string) {
	fmt.Println()
	fmt.Println(theme.ErrorStyle.Render("✗"), theme.Primary.Render(title))
	fmt.Println()
	if action1 != "" {
		fmt.Printf("  %s %s\n", theme.Muted.Render("→"), action1)
	}
	if action2 != "" {
		fmt.Printf("  %s %s\n", theme.Muted.Render("→"), action2)
	}
	if action3 != "" {
		fmt.Printf("  %s %s\n", theme.Muted.Render("→"), action3)
	}
	fmt.Println()
}

func ErrorWithHint(title string, hints []string) {
	fmt.Println()
	fmt.Println(theme.ErrorStyle.Render("✗"), theme.Primary.Render(title))
	fmt.Println()
	for _, hint := range hints {
		fmt.Printf("  %s %s\n", theme.Muted.Render("→"), hint)
	}
	fmt.Println()
}

func ServerUnreachable(host string) {
	ActionableError(
		fmt.Sprintf("cannot reach khayal at %s", host),
		"is khayal running?",
		"wrong address?        kl config set host <address>",
		"check logs in khayal",
	)
}

func AuthFailed() {
	ActionableError(
		"unauthorized · invalid token",
		"get token from        ~/.config/khayal/config.yaml on server",
		"update token          kl config set token <token>",
		"",
	)
}

func VaultWriteFailed() {
	ActionableError(
		"cannot write to vault",
		"check permissions     ls -la <vault>/<inbox_dir>/",
		"vault path correct?   khayal config",
		"",
	)
}

func DepMissing(dep string) {
	ActionableError(
		fmt.Sprintf("missing dependency: %s", dep),
		fmt.Sprintf("install with:       brew install %s", dep),
		"",
		"",
	)
}
