package plage

import (
	"context"
	"flag"
	"fmt"
	"os"
)

var smFlag = flag.String("age-plugin", "", "The state machine selector flag as per the age-plugin spec. Currently supports 'recipient-v1' and 'identity-v1'")

func init() {
	flag.Parse()
}

//func GetMachine(name string) any {
//	switch name {
//	case "recipient-v1":
//		return newRecipientV1
//	case "identity-v1":
//		return identityV1
//	}
//
//	fmt.Fprintln(os.Stderr, "invalid state machine")
//	os.Exit(1)
//}

type RecipientV1 interface {
	WrapFileKeyFromRecipients([]byte, [][]byte) []byte
	WrapFileKeyFromIdentity([]byte, [][]byte) []byte
	GetName() string
}

type smRecipientV1 struct {
	recipients [][]byte
	identities [][]byte
	fileKeys   [][]byte
}

func (r *smRecipientV1) addRecipient(c *Command, plugin *RecipientV1) {
	name, rec, err := agep.ParseRecipient(c.Metadata())
	if err != nil || name != plugin.GetName() {
		return
	}

	r.recipients = append(r.recipients, rec)
}

func (r *smRecipientV1) addIdentity(c *Command, plugin *RecipientV1) {
	name, sec, err := agep.ParseIdentify(c.Metadata())
	if err != nil || name != plugin.GetName() {
		return
	}

	r.identities = append(r.recipients, rec)
}

func (r *smRecipientV1) addFileKey(c *Command) {
	r.fileKeys = append(r.fileKeys, c.Data())
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
			r.addFileKey(cmd)
		case Done:
			return r.doIt(plugin)
		default:
			continue
		}
	}
}
