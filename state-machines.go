package plage

import (
	"flag"
	"fmt"
	"os"

	"filippo.io/age"
)

var smFlag = flag.String("age-plugin", "", "The state machine selector flag as per age-plugin spec")

func init() {
	flag.Parse()
}

func GetMachine(name string) any {
	switch name {
	case "recipient-v1":
		return recipientV1
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

func (r *recipientV1) addRecipient(c *Command) {

}

func (r *recipientV1) addIdentity(c *Command) {

}

func (r *recipientV1) wrapFileKey(c *Command) {

}

func (r *recipientV1) Phase1(c *CmdReader) {
	errCount := 0
	for errCount < 5 {
		cmd, err := c.ReadCommand()
		if err != nil {
			errCount++
			continue
		}
		switch cmd {
		case nil:
			continue
		case cmd.Name() == "add-recipient":
			addRecipient(cmd)
		case cmd.Name() == "add-identity":
			addIdentity(cmd)
		case cmd.Name() == "wrap-file-key":
			wrapFileKey(cmd)
		case Done:
			DoIt()
		default:
			continue
		}
	}
	fmt.Fprintln(os.Stderr, "Too many errors while parsing commands in Recipient V1 Phase 1, aborting")
}
