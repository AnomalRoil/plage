package plage

import (
	"context"
	"flag"
	"fmt"
	"os"

	"filippo.io/age"
)

var smFlag = flag.String("age-plugin", "", "The state machine selector flag as per the age-plugin spec. Currently supports 'recipient-v1' and 'identity-v1'")

func init() {
	flag.Parse()
}

func GetMachine(name string) any {
	switch name {
	case "recipient-v1":
		return newRecipientV1
	case "identity-v1":
		return identityV1
	}

	fmt.Fprintln(os.Stderr, "invalid state machine")
	os.Exit(1)
}

type RecipientV1 interface {
	ParseRecipient([]byte) age.Recipient
	ParseIdentify([]byte) age.Identity
	WrapFileKey([]byte) []byte
}

type smRecipientV1 struct {
	recipients []age.Recipient
	identities []age.Identity
}

func (r *smRecipientV1) addRecipient(c *Command) {

}

func (r *smRecipientV1) addIdentity(c *Command) {

}

func (r *smRecipientV1) wrapFileKey(c *Command) {

}

func (r *smRecipientV1) doIt() error {

}

func (r *smRecipientV1) Phase1(ctx context.Context, c *CmdReader, plugin *RecipientV1) error {
	errCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		cmd, err := c.ReadCommand()
		if err != nil {
			errCount++
			fmt.Fprintln(os.Stderr, "error reading command:", err, errCount)
			continue
		}
		switch cmd {
		case nil:
			continue
		case cmd.Name() == "add-recipient":
			r.addRecipient(cmd, plugin)
		case cmd.Name() == "add-identity":
			r.addIdentity(cmd, plugin)
		case cmd.Name() == "wrap-file-key":
			r.wrapFileKey(cmd, plugin)
		case Done:
			return r.doIt()
		default:
			continue
		}
	}
}
