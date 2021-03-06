package handler

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/howeyc/gopass"
	"github.com/ory-am/hydra/account"
	"github.com/pborman/uuid"
)

type Account struct {
	Ctx *Context
}

func getPassword() (password string) {
	fmt.Println("Password: ")
	pwd, err := gopass.GetPasswd()
	if err != nil {
		fmt.Errorf("Error: %s", err)
		return getPassword()
	}
	password = string(pwd);
	if string(pwd) == "" {
		fmt.Println("You did not provide a password. Please try again.")
		return getPassword()
	}

	fmt.Println("Confirm password: ")
	confirm, err := gopass.GetPasswd()
	if err != nil {
		fmt.Errorf("Error: %s", err)
		return getPassword()
	}
	if password != string(confirm) {
		fmt.Println("Password and confirmation do not match. Please try again.")
		return getPassword()
	}
	return
}

func (c *Account) Create(ctx *cli.Context) error {
	username := ctx.Args().First()
	if username == "" {
		return fmt.Errorf("Please provide an username.")
	}
	password := ctx.String("password")
	if password == "" {
		password = getPassword()
	}

	c.Ctx.Start()
	user, err := c.Ctx.Accounts.Create(account.CreateAccountRequest{
		ID:       uuid.New(),
		Username: username,
		Password: password,
		Data:     "{}",
	})
	if err != nil {
		return fmt.Errorf("Could not create account because %s", err)
	}

	fmt.Printf(`Created account as "%s".`+"\n", user.GetID())
	if ctx.Bool("as-superuser") {
		if err := c.Ctx.Policies.Create(superUserPolicy(user.GetID())); err != nil {
			return fmt.Errorf("Could not create policy for account because %s", err)
		}
		fmt.Printf(`Granted superuser privileges to account "%s".`+"\n", user.GetID())
	}
	return nil
}
