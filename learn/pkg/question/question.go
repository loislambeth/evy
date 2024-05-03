// Package question provides data structures and tools for Evy course
// questions. Question are parsed from question Markdown files with YAML
// frontmatter. The frontmatter serves as a small set of structured data
// associated with the unstructured Markdown file.
//
// Question can be verified to have the expected correct answer output match
// the question output. Questions, can seal (encrypt) their answers in the
// Frontmatter or unsealed (decrypted) them. We use this to avoid openly
// publishing the answerKey. Questions can also export their AnswerKeys into
// single big JSON object as used in Evy's persistent data store(Firestore).
// See the testdata/ directory for sample question and answers.
package question

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
	"rsc.io/markdown"
)

// Errors for the question package.
var (
	ErrBadMarkdownStructure = errors.New("bad Markdown structure")
	ErrInconsistentMdoel    = errors.New("inconsistency")
	ErrWrongAnswer          = errors.New("wrong answer")

	ErrSingleChoice          = errors.New("single-choice answer must be a single character a-z")
	ErrBadDirectoryStructure = errors.New("bad directory structure for course layout")

	ErrNoFrontmatter        = errors.New("no frontmatter found")
	ErrInvalidFrontmatter   = errors.New("invalid frontmatter")
	ErrWrongFrontmatterType = errors.New("wrong frontmatter type")
	ErrNoFrontmatterAnswer  = errors.New("no answer in frontmatter")
	ErrSealedAnswerNoKey    = errors.New("sealed answer without key in frontmatter")
	ErrSealedTooShort       = errors.New("sealed data is too short")
)

// Markdown is a markdown file with question frontmatter.
type Markdown struct {
	Filename    string
	Frontmatter *frontmatter
	Doc         *markdown.Document
}

// NewMarkdown creates a new Markdown value for the file with given filename.
// Markdown contains the parsed frontmatter and the markdown AST.
func NewMarkdown(filename string) (*Markdown, error) {
	frontmatterString, mdString, err := readSplitMDFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%w (%s)", err, filename)
	}
	fm, err := parseFrontmatter(frontmatterString)
	if err != nil {
		return nil, fmt.Errorf("%w (%s)", err, filename)
	}
	parser := markdown.Parser{AutoLinkText: true, TaskListItems: true}
	doc := parser.Parse(mdString)

	return &Markdown{Filename: filename, Frontmatter: fm, Doc: doc}, nil
}

// Seal seals the unsealed answer in the Frontmatter using the public key.
// Sealing can only be reverted if the secret private key is available.
// Answers committed to the public Repo should always be sealed.
func (md *Markdown) Seal(publicKey string) error {
	if err := md.Frontmatter.Seal(publicKey); err != nil {
		return fmt.Errorf("%w (%s)", err, md.Filename)
	}
	return nil
}

// Unseal unseals the sealed answer in the Frontmatter using the private key.
func (md *Markdown) Unseal(privateKey string) error {
	if err := md.Frontmatter.Unseal(privateKey); err != nil {
		return fmt.Errorf("%w (%s)", err, md.Filename)
	}
	return nil
}

// Format formats YAML frontmatter, fenced by "---", followed by markdown
// content.
func (md *Markdown) Format() (string, error) {
	bytes, err := yaml.Marshal(md.Frontmatter)
	if err != nil {
		return "", err
	}
	sb := strings.Builder{}
	sb.WriteString("---\n")
	sb.Write(bytes)
	sb.WriteString("---\n\n")
	sb.WriteString(markdown.ToMarkdown(md.Doc))
	return sb.String(), nil
}

// Verify checks that only the correct answer choice (output) matches the
// question (output). "Correct" means as specified in the Markdown
// Frontmatter.
func (md *Markdown) Verify(key string) error {
	_, err := md.getVerifiedAnswer(key)
	return err
}

// VerifyUnsealed checks unsealed answers only and ignores sealed ones.
// For unsealed answers it performs a normal Verify.
func (md *Markdown) VerifyUnsealed() error {
	if md.Frontmatter.SealedAnswer != "" {
		return nil
	}
	_, err := md.getVerifiedAnswer("")
	return err
}

// ExportAnswerKey returns the answerKey for the question Markdown file.
func (md *Markdown) ExportAnswerKey(key string) (AnswerKey, error) {
	answer, err := md.getVerifiedAnswer(key)
	if err != nil {
		return nil, err
	}
	return NewAnswerKey(md.Filename, answer)
}

// ExportAnswerKeyUnsealed returns the answerKey for the question Markdown
// file if the answer is unsealed. It returns an empty AnswerKey otherwise.
func (md *Markdown) ExportAnswerKeyUnsealed() (AnswerKey, error) {
	if md.Frontmatter.SealedAnswer != "" {
		return AnswerKey{}, nil
	}
	answer, err := md.getVerifiedAnswer("")
	if err != nil {
		return nil, err
	}
	return NewAnswerKey(md.Filename, answer)
}

func (md *Markdown) getVerifiedAnswer(key string) (Answer, error) {
	answer, err := md.Frontmatter.getAnswer(key)
	if err != nil {
		return Answer{}, fmt.Errorf("%w (%s)", err, md.Filename)
	}
	model, err := NewModel(md)
	if err != nil {
		return Answer{}, fmt.Errorf("%w (%s)", err, md.Filename)
	}
	if err := model.Verify(answer); err != nil {
		return Answer{}, fmt.Errorf("%w (%s)", err, md.Filename)
	}
	return answer, nil
}

// readSplitMDFile returns contents of filename split into frontmatter and
// markdown string.
func readSplitMDFile(filename string) (string, string, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return "", "", fmt.Errorf("cannot process Question Markdown file: %w", err)
	}
	str := trimLeftSpace(string(b))
	frontmatter, err := extractFrontmatterString(str)
	if err != nil {
		return "", "", err
	}
	md := trimLeftSpace(str[len(frontmatter)+6:])
	return frontmatter, md, nil
}

func trimLeftSpace(str string) string {
	return strings.TrimLeftFunc(str, unicode.IsSpace)
}
