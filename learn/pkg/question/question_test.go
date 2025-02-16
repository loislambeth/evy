package question

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestNewModel(t *testing.T) {
	for name := range testQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			got, err := NewModel(fname)
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
			model, err := NewModel(fname)
			assert.NoError(t, err)
			err = model.Verify()
			assert.NoError(t, err)
		})
	}
}

func TestExportAnswer(t *testing.T) {
	for name, want := range testQuestions {
		t.Run(name, func(t *testing.T) {
			b, err := os.ReadFile("testdata/golden/answerkey-" + name + ".json")
			assert.NoError(t, err)
			wantAnswerKey := AnswerKey{}
			err = json.Unmarshal(b, &wantAnswerKey)
			assert.NoError(t, err)

			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			model, err := NewModel(fname)
			assert.NoError(t, err)
			gotAnswerKey, err := model.ExportAnswerKey()
			assert.NoError(t, err)

			model, err = NewModel(fname, WithPrivateKey(testKeyPrivate))
			assert.NoError(t, err)
			gotAnswerKeyWithSealed, err := model.ExportAnswerKey()
			assert.NoError(t, err)

			assert.Equal(t, wantAnswerKey, gotAnswerKey)
			assert.Equal(t, wantAnswerKey, gotAnswerKeyWithSealed)

			got := gotAnswerKey["course1"]["unit1"]["exercise1"][name]
			assert.Equal(t, true, want.Equals(got))
		})
	}
}

func TestSealAnswer(t *testing.T) {
	for name, answer := range testQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			model, err := NewModel(fname)
			assert.NoError(t, err)

			err = model.Seal(testKeyPublic)
			assert.NoError(t, err)
			assert.Equal(t, "", model.Frontmatter.Answer)
			unsealedAnswer, err := Decrypt(testKeyPrivate, model.Frontmatter.SealedAnswer)
			assert.NoError(t, err)
			want := answer.correctAnswers()
			assert.Equal(t, want, unsealedAnswer)
		})
	}
}

func TestUnsealAnswer(t *testing.T) {
	fname := "testdata/course1/unit1/exercise1/questions/question1-sealed.md"
	model, err := NewModel(fname, WithPrivateKey(testKeyPrivate))
	assert.NoError(t, err)
	assert.Equal(t, "", model.Frontmatter.Answer)

	err = model.Unseal()
	assert.NoError(t, err)
	assert.Equal(t, "c", model.Frontmatter.Answer)
	assert.Equal(t, "", model.Frontmatter.SealedAnswer)
}

func TestExportAnswerKeyFromSeal(t *testing.T) {
	fname := "testdata/course1/unit1/exercise1/questions/question1-sealed.md"
	model, err := NewModel(fname, WithPrivateKey(testKeyPrivate))
	assert.NoError(t, err)
	gotAnswerKey, err := model.ExportAnswerKey()
	assert.NoError(t, err)

	b, err := os.ReadFile("testdata/golden/answerkey-question1-sealed.json")
	assert.NoError(t, err)
	wantAnswerKey := AnswerKey{}
	err = json.Unmarshal(b, &wantAnswerKey)
	assert.NoError(t, err)
	assert.Equal(t, wantAnswerKey, gotAnswerKey)

	want := Answer{Single: "c"}
	got := gotAnswerKey["course1"]["unit1"]["exercise1"]["question1-sealed"]
	assert.Equal(t, true, want.Equals(got))

	model, err = NewModel(fname) // no key
	assert.NoError(t, err)
	gotAnswerKey, err = model.ExportAnswerKey()
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
			fname := "testdata/course1/unit1/err-exercise1/questions/" + name + ".md"
			model, err := NewModel(fname)
			assert.NoError(t, err)
			err = model.Verify()
			assert.Error(t, ErrWrongAnswer, err)
		})
	}
}

func TestErrNoExistMD(t *testing.T) {
	fname := "testdata/course1/unit1/exercise1/questions/MISSING-FILE.md"
	_, err := NewModel(fname)
	assert.Error(t, os.ErrNotExist, err)
}

func TestErrNoExistSVG(t *testing.T) {
	fname := "testdata/course1/unit1/err-exercise1/questions/err-img3.md"
	_, err := NewModel(fname)
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
			fname := "testdata/course1/unit1/err-exercise1/questions/" + name + ".md"
			_, err := NewModel(fname)
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
			fname := "testdata/course1/unit1/err-exercise1/questions/" + name + ".md"
			_, err := NewModel(fname)
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
			fname := "testdata/course1/unit1/err-exercise1/questions/" + name + ".md"
			_, err := NewModel(fname)
			assert.Error(t, ErrInconsistentMdoel, err)
		})
	}
}

func TestRendererTracking(t *testing.T) {
	embeds := map[string][]string{
		"question1":      nil,
		"question2":      nil,
		"question-img1":  {"dot-dot-a-evy-svg", "dot-dot-b-evy-svg", "dot-dot-c-evy-svg", "dot-dot-d-evy-svg"},
		"question-img2":  {"dot-dot-a-evy-svg"},
		"question-link1": {"dot-dot-c-evy", "dot-dot-a-evy", "dot-dot-b-evy", "dot-dot-c-evy", "dot-dot-d-evy"},
		"question-link2": {"dot-dot-c-evy", "dot-dot-a-evy", "dot-dot-b-evy", "dot-dot-c-evy", "dot-dot-d-evy"},
		"question-link3": {"print-print-b-evy", "print-print-a-evy", "print-print-b-evy", "print-print-c-evy", "print-print-d-evy"},
		"question-link4": {"print-print-d-evy", "print-print-a-evy", "print-print-b-evy", "print-print-c-evy", "print-print-d-evy"},
	}
	for name, want := range embeds {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			model, err := NewModel(fname)
			assert.NoError(t, err)
			got := model.embeds
			assert.Equal(t, len(want), len(got))
			m := map[string]bool{}
			for _, w := range want {
				m[w] = true
			}
			for _, g := range got {
				assert.Equal(t, true, m[g.id], "cannot find "+g.id)
			}
		})
	}
}

func TestPrintHTML(t *testing.T) {
	for name := range testQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			model, err := NewModel(fname)
			assert.NoError(t, err)
			buf := &bytes.Buffer{}
			model.PrintHTML(buf)
			got := buf.String()

			goldenFile := filepath.Join("testdata/golden/", "form-"+name+".html")
			b, err := os.ReadFile(goldenFile)
			assert.NoError(t, err)
			want := string(b)
			assert.Equal(t, want, got)
		})
	}
}

func TestToHTML(t *testing.T) {
	for name := range testQuestions {
		t.Run(name, func(t *testing.T) {
			fname := "testdata/course1/unit1/exercise1/questions/" + name + ".md"
			model, err := NewModel(fname)
			assert.NoError(t, err)
			got := model.ToHTML()

			goldenFile := filepath.Join("testdata/golden/", name+".html")
			b, err := os.ReadFile(goldenFile)
			assert.NoError(t, err)
			want := string(b)
			assert.Equal(t, want, got)
		})
	}
}
