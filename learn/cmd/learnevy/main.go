// Learnevy is a tool for creating Evy practice and learn materials.
//
// Learnevy has the following sub-commands: export, verify, serve, seal, unseal.
//
//	Usage: learnevy <command> [flags]
//
//	learnevy is a tool that manages practice and learn resources for Evy.
//
//	Flags:
//	  -h, --help       Show context-sensitive help.
//	  -V, --version    Print version information
//
//	Commands:
//	  export <md-file> [<answer-key-file>] [flags]
//	    Export answer key File.
//
//	  verify <md-file> [<type>] [flags]
//	    Verify encryptedAnsers in markdown file. Ensure no plaintext answers.
//
//	  seal <md-file> [flags]
//	    Move 'answer' to 'sealed-answer' in source markdown.
//
//	  unseal <md-file> [flags]
//	    Move 'sealed-answer' to 'answer' in source markdown.
//
//	Run "learnevy <command> --help" for more information on a command.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"evylang.dev/evy/learn/pkg/question"
	"github.com/alecthomas/kong"
)

const description = `
learnevy is a tool that manages practice and learn resources for Evy.
`

var version = "v0.0.0"

type app struct {
	Export exportCmd `cmd:"" help:"Export answer key File."`
	Verify verifyCmd `cmd:"" help:"Verify encryptedAnsers in markdown file. Ensure no plaintext answers."`

	Seal   sealCmd   `cmd:"" help:"Move 'answer' to 'sealed-answer' in source markdown."`
	Unseal unsealCmd `cmd:"" help:"Move 'sealed-answer' to 'answer' in source markdown."`

	Version kong.VersionFlag `short:"V" help:"Print version information"`

	Crypto cryptoCmd `cmd:"" help:"Encryption utilities." hidden:""`
}

type cryptoCmd struct {
	Keygen keygenCryptoCmd `cmd:"" help:"Generate a new secret key."`
	Seal   sealCryptoCmd   `cmd:"" help:"Encrypt a string given on command line"`
	Unseal unsealCryptoCmd `cmd:"" help:"Decrypt string given on command line"`
}

func main() {
	kopts := []kong.Option{
		kong.Description(description),
		kong.Vars{"version": version},
	}
	kctx := kong.Parse(&app{}, kopts...)
	kctx.FatalIfErrorf(kctx.Run())
}

type keygenCryptoCmd struct {
	Length int `short:"l" default:"2048" help:"Length of key to generate."`
}
type sealCryptoCmd struct {
	Plaintext string `arg:"" help:"Plaintext to encrypt."`
}
type unsealCryptoCmd struct {
	Ciphertext string `arg:"" help:"Ciphertext to decrypt."`
	PrivateKey string `short:"s" help:"Secret private key to decrypt ciphertext." env:"EVY_LEARN_PRIVATE_KEY"`
}

func (c *keygenCryptoCmd) Run() error {
	keys, err := question.Keygen(c.Length)
	if err != nil {
		return err
	}
	fmt.Printf("private: %s\n\npublic:  %s\n", keys.Private, keys.Public)
	return nil
}

func (c *sealCryptoCmd) Run() error {
	encrypted, err := question.Encrypt(question.PublicKey, c.Plaintext)
	if err != nil {
		return err
	}
	fmt.Println(encrypted)
	return nil
}

func (c *unsealCryptoCmd) Run() error {
	plaintext, err := question.Decrypt(c.PrivateKey, c.Ciphertext)
	if err != nil {
		return err
	}
	fmt.Println(plaintext)
	return nil
}

type exportCmd struct {
	MDFile        string `arg:"" type:"markdownfile" help:"Markdown file with course, unit, exercise, or question." placeholder:"ANSWERFILE"`
	AnswerKeyFile string `arg:"" default:"-" help:"JSON output file for answer key (default: stdout)." placeholder:"ANSWERFILE"`
	UnsealedOnly  bool   `short:"u" help:"Only export files with unsealed answers. Suitable if private key not available."`
	PrivateKey    string `short:"k" help:"Secret private key to decrypt sealed answers." env:"EVY_LEARN_PRIVATE_KEY"`
}

type sealCmd struct {
	MDFile    string `arg:"" type:"markdownfile" help:"Markdown file with course, unit, exercise, or question." placeholder:"ANSWERFILE"`
	PublicKey string `short:"k" help:"public key to seal answers, default provided"`
}

type unsealCmd struct {
	MDFile     string `arg:"" type:"markdownfile" help:"Markdown file with course, unit, exercise, or question." placeholder:"ANSWERFILE"`
	PrivateKey string `short:"k" help:"Secret private key to decrypt sealed answers." env:"EVY_LEARN_PRIVATE_KEY"`
}

type verifyCmd struct {
	MDFile       string `arg:"" type:"markdownfile" help:"Markdown file with course, unit, exercise, or question." placeholder:"ANSWERFILE"`
	Type         string `arg:"" default:"all" enum:"all,result,seal" help:"Type of verification to perform."`
	UnsealedOnly bool   `short:"u" help:"Only check result for files with unsealed answers. Suitable if private key not available."`
	PrivateKey   string `short:"k" help:"Secret private key to decrypt sealed answers." env:"EVY_LEARN_PRIVATE_KEY"`
}

func (c *exportCmd) Run() error {
	md, err := question.NewMarkdown(c.MDFile)
	if err != nil {
		return err
	}
	var answerKey question.AnswerKey
	if c.UnsealedOnly {
		answerKey, err = md.ExportAnswerKeyUnsealed()
	} else {
		answerKey, err = md.ExportAnswerKey(c.PrivateKey)
	}
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(answerKey, "", "  ")
	if err != nil {
		return err
	}
	if c.AnswerKeyFile != "-" {
		return os.WriteFile(c.AnswerKeyFile, append(b, '\n'), 0o666)
	}
	fmt.Println(string(b))
	return nil
}

func (c *sealCmd) Run() error {
	md, err := question.NewMarkdown(c.MDFile)
	if err != nil {
		return err
	}
	publicKey := c.PublicKey
	if publicKey == "" {
		publicKey = question.PublicKey
	}
	if err := md.Seal(publicKey); err != nil {
		return err
	}
	formatted, err := md.Format()
	if err != nil {
		return err
	}
	return os.WriteFile(c.MDFile, []byte(formatted), 0o666)
}

func (c *unsealCmd) Run() error {
	md, err := question.NewMarkdown(c.MDFile)
	if err != nil {
		return err
	}
	if err := md.Unseal(c.PrivateKey); err != nil {
		return err
	}
	formatted, err := md.Format()
	if err != nil {
		return err
	}
	return os.WriteFile(c.MDFile, []byte(formatted), 0o666)
}

func (c *verifyCmd) Run() error {
	md, err := question.NewMarkdown(c.MDFile)
	if err != nil {
		return err
	}
	if c.UnsealedOnly {
		return md.VerifyUnsealed()
	}
	return md.Verify(c.PrivateKey)
}
