package question

import (
	"encoding/json"
	"os"
	"testing"

	"evylang.dev/evy/pkg/assert"
)

var testQuestions = map[string]Answer{
	"question1":      {Single: "c"},
	"question2":      {Single: "a"},
	"question-img1":  {Single: "b"},
	"question-img2":  {Single: "d"},
	"question-link1": {Single: "c"},
	"question-link2": {Single: "c"},
	"question-link3": {Single: "d"},
	"question-link4": {Multi: []string{"b", "c"}},
}

func TestNewMarkdown(t *testing.T) {
	for name := range testQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			got, err := NewMarkdown(fname)
			assert.NoError(t, err)

			assert.Equal(t, fname, got.Filename)
			want := frontmatterType("question")
			assert.Equal(t, want, got.Frontmatter.Type)
		})
	}
}

func TestValidateAnswer(t *testing.T) {
	for name := range testQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			md, err := NewMarkdown(fname)
			assert.NoError(t, err)
			err = md.Verify("")
			assert.NoError(t, err)
		})
	}
}

func TestExportAnswer(t *testing.T) {
	for name, want := range testQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			md, err := NewMarkdown(fname)
			assert.NoError(t, err)
			privateKey := ""
			gotAnswerKey, err := md.ExportAnswerKey(privateKey)
			assert.NoError(t, err)
			gotAnswerKeyUnselaed, err := md.ExportAnswerKeyUnsealed()
			assert.NoError(t, err)

			b, err := os.ReadFile("testdata/golden/answerkey-" + name + ".json")
			assert.NoError(t, err)
			wantAnswerKey := AnswerKey{}
			err = json.Unmarshal(b, &wantAnswerKey)
			assert.NoError(t, err)
			assert.Equal(t, wantAnswerKey, gotAnswerKey)
			assert.Equal(t, wantAnswerKey, gotAnswerKeyUnselaed)

			got := gotAnswerKey["course1"]["unit1"]["exercise1"][name]
			assert.Equal(t, true, want.Equals(got))
		})
	}
}

func TestSealAnswer(t *testing.T) {
	for name, answer := range testQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			md, err := NewMarkdown(fname)
			assert.NoError(t, err)

			err = md.Seal(testKeyPublic)
			assert.NoError(t, err)
			assert.Equal(t, "", md.Frontmatter.Answer)
			unsealedAnswer, err := Decrypt(testKeyPrivate, md.Frontmatter.SealedAnswer)
			assert.NoError(t, err)
			want := answer.correctAnswers()
			assert.Equal(t, want, unsealedAnswer)
		})
	}
}

func TestUnsealAnswer(t *testing.T) {
	fname := "testdata/course1/unit1/exercise1/questions/question1-sealed.md"
	md, err := NewMarkdown(fname)
	assert.NoError(t, err)
	assert.Equal(t, "", md.Frontmatter.Answer)

	err = md.Unseal(testKeyPrivate)
	assert.NoError(t, err)
	assert.Equal(t, "c", md.Frontmatter.Answer)
	assert.Equal(t, "", md.Frontmatter.SealedAnswer)
}

func TestExportAnswerKeyFromSeal(t *testing.T) {
	fname := "testdata/course1/unit1/exercise1/questions/question1-sealed.md"
	md, err := NewMarkdown(fname)
	assert.NoError(t, err)
	gotAnswerKey, err := md.ExportAnswerKey(testKeyPrivate)
	assert.NoError(t, err)

	b, err := os.ReadFile("testdata/golden/answerKey-question1-sealed.json")
	assert.NoError(t, err)
	wantAnswerKey := AnswerKey{}
	err = json.Unmarshal(b, &wantAnswerKey)
	assert.NoError(t, err)
	assert.Equal(t, wantAnswerKey, gotAnswerKey)

	want := Answer{Single: "c"}
	got := gotAnswerKey["course1"]["unit1"]["exercise1"]["question1-sealed"]
	assert.Equal(t, true, want.Equals(got))

	gotAnswerKey, err = md.ExportAnswerKeyUnsealed()
	assert.NoError(t, err)
	assert.Equal(t, AnswerKey{}, gotAnswerKey)
}

func TestErrInvalidAnswer(t *testing.T) {
	errQuestions := []string{
		"err-false-positive",
		"err-false-negative",
		"err-img1",
		"err-img2",
	}
	for _, name := range errQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			md, err := NewMarkdown(fname)
			assert.NoError(t, err)
			err = md.Verify("")
			assert.Error(t, ErrWrongAnswer, err)
		})
	}
}

func TestErrNoExistMD(t *testing.T) {
	fname := "testdata/course1/unit1/exercise1/questions/MISSING-FILE.md"
	_, err := NewMarkdown(fname)
	assert.Error(t, os.ErrNotExist, err)
}

func TestErrNoExistSVG(t *testing.T) {
	fname := "testdata/course1/unit1/exercise1/questions/err-img3.md"
	md, err := NewMarkdown(fname)
	assert.NoError(t, err)
	err = md.Verify("")
	assert.Error(t, os.ErrNotExist, err)
}

func TestErrBadMDImg(t *testing.T) {
	errQuestions := []string{
		"err-img4",
		"err-img5",
		"err-img6",
		"err-img7",
	}
	for _, name := range errQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			md, err := NewMarkdown(fname)
			assert.NoError(t, err)
			err = md.Verify("")
			assert.Error(t, ErrBadMarkdownStructure, err)
		})
	}
}

func TestErrBadMDLink(t *testing.T) {
	errQuestions := []string{
		"err-link1",
		"err-link2",
		"err-link3",
		"err-link4",
		"err-link5",
		"err-link6",
	}
	for _, name := range errQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			md, err := NewMarkdown(fname)
			assert.NoError(t, err)
			err = md.Verify("")
			assert.Error(t, ErrBadMarkdownStructure, err)
		})
	}
}

func TestErrInconsistency(t *testing.T) {
	errQuestions := []string{
		"err-inconsistent1",
		"err-inconsistent2",
	}
	for _, name := range errQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			md, err := NewMarkdown(fname)
			assert.NoError(t, err)
			err = md.Verify("")
			assert.Error(t, ErrInconsistentMdoel, err)
		})
	}
}
