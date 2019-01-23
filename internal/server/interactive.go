package server

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"gitlab.com/agubarev/hometown/usermanager"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

func interactiveCreateSuperDomain(m *usermanager.UserManager) error {
	// InteractiveCreateSuperDomain prompts a user for details when running
	// the server instance for the first time
	qs := []*survey.Question{
		{
			Name:   "username",
			Prompt: &survey.Input{Message: "Enter username of a super user:"},
		},
		{
			Name:   "email",
			Prompt: &survey.Input{Message: "Enter email:"},
		},
	}

	// answers
	as := struct {
		Username string
		Email    string
	}{}

	// scanning answers
	if err := survey.Ask(qs, &as); err != nil {
		return fmt.Errorf("failed to create super domain: %s", err)
	}

	// initializing superuser object
	superuser, err := usermanager.NewUser(as.Username, as.Email)
	if err != nil {
		return err
	}

	// validating superuser object
	if err := superuser.Validate(); err != nil {
		return fmt.Errorf("superuser validation failed: %s", err)
	}

	// now superuser needs a password
	pqs := []*survey.Question{
		{
			Name:   "password1",
			Prompt: &survey.Password{Message: "Enter new password:"},
		},
		{
			Name:   "password2",
			Prompt: &survey.Password{Message: "Re-enter new password to confirm:"},
		},
	}

	// password answers
	pas := struct {
		Password1 string
		Password2 string
	}{}

	// scanning passwords until both passwords match
	var password string
	for {
		if err := survey.Ask(pqs, &pas); err != nil {
			return fmt.Errorf("password entry failed: %s", err)
		}

		if pas.Password1 == pas.Password2 {
			password = pas.Password2
			// breaking from the loop
			break
		}
	}

	// creating new super domain with this new user as its superuser owner
	err = m.CreateSuperDomain(superuser)
	if err != nil {
		return fmt.Errorf("failed to create super domain: %s", err)
	}

	spew.Dump(superuser)

	// setting password for superuser
	err = m.SetUserPassword(superuser, password)
	if err != nil {
		return fmt.Errorf("failed to assign password for superuser: %s", err)
	}

	return nil
}
