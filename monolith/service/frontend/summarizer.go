package frontend

import (
	"bufio"
	"bytes"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type matchedSentence struct {
	// Location / position of the sentence in the entire page text / content.
	position int

	// sentence text.
	text string

	// Ratio of matched keywords to the total number of words in the sentence.
	matchRatio float32
}

type matchSummarizer struct {
	// List of terms in a search query.
	searchTerms []string

	// Maximum search summary size represented as a number of characters.
	maxSummaryLen int

	// Re-usable buffer for generating a summary.
	sumBuff bytes.Buffer
}

func newMatchSummarizer(searchTerms string, maxSummaryLen int) *matchSummarizer {
	return &matchSummarizer{
		searchTerms:   strings.Fields(strings.Trim(searchTerms, `"`)),
		maxSummaryLen: maxSummaryLen,
	}
}

// Summary formats and returns a summary string of the matched search terms.
func (s *matchSummarizer) Summary(content string) string {
	s.sumBuff.Reset()

	// Last sentence position.
	lastPosition := -1
	for _, sentence := range s.sentencesForSummary(content) {
		if lastPosition != -1 && sentence.position-lastPosition != 1 {
			_, _ = s.sumBuff.WriteString("...")
		}

		lastPosition = sentence.position

		_, _ = s.sumBuff.WriteString(sentence.text)

		if !strings.HasSuffix(sentence.text, ".") {
			_ = s.sumBuff.WriteByte('.')
		}
	}

	return strings.TrimSpace(s.sumBuff.String())
}

// sentencesForSummary splits the page content into sentences and keeps those
// with atleast one matching search term.
func (s *matchSummarizer) sentencesForSummary(content string) []*matchedSentence {
	var matched []*matchedSentence

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Split(scanSentence)

	// Scan and spit the page content into individual sentences.
	for position := 0; scanner.Scan(); position++ {
		sentence := scanner.Text()
		if matchRatio := s.matchRatio(sentence); matchRatio > 0 {
			matched = append(matched, &matchedSentence{
				position:   position,
				text:       sentence,
				matchRatio: matchRatio,
			})
		}
	}

	// Sort by match ratio in descending order (higher quality matches first). This
	// ensure that the higher quality matches take up the summary space first in case
	// of character space limits.
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].matchRatio > matched[j].matchRatio
	})

	// Select matches from the sorted list until we exhaust the max summary length.
	var summary []*matchedSentence

	for i, remainingLen := 0, s.maxSummaryLen; i < len(matched) && remainingLen > 0; i++ {
		// If we cannot fit the entire sentence, trim its end. Note: this may
		// result in summaries that do not contain any of the keywords.
		if sentenceLen := len(matched[i].text); sentenceLen > remainingLen {
			matched[i].text = string(([]rune(matched[i].text))[:remainingLen]) + "..."
		}

		remainingLen -= len(matched[i].text)
		summary = append(summary, matched[i])
	}

	// Sort selected summary sentences by position in ascending order. This ensures that
	// the sentences in the summary have the same order as the original document.
	sort.Slice(summary, func(i, j int) bool {
		return summary[i].position < summary[j].position
	})

	return summary
}

// matchRatio computes and returns the ratio of matched search terms to total words
// in a sentence.
func (s *matchSummarizer) matchRatio(sentence string) float32 {
	var wordCount, matchedWordCount int

	scanner := bufio.NewScanner(strings.NewReader(sentence))
	scanner.Split(bufio.ScanWords)

	for ; scanner.Scan(); wordCount++ {
		word := scanner.Text()
		for _, term := range s.searchTerms {
			if strings.EqualFold(term, word) {
				matchedWordCount++

				break
			}
		}
	}

	// Avoid division by zero errors.
	if wordCount == 0 {
		wordCount = 1
	}

	return float32(matchedWordCount) / float32(wordCount)
}

// This serves as a [bufio.scanner.SpitFunc]. it scans and splits text based on
// specific termination characters and symbols. ie ['!', '?', '.'].
func scanSentence(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF {
		if len(data) == 0 {
			return 0, nil, nil
		}

		return len(data), data, nil
	}

	var seq [3]rune
	var index, skip int

	for i := 0; i < len(seq); i++ {
		if seq[i], skip = scanRune(data[index:]); skip < 0 {
			return 0, nil, nil
		}
		// Update the index with the byte size of the returned rune. This ensures that
		// the correct bytes are accessed on the following iterations.
		// Runes that represent ASCII chars have a byte size of 1, while other char
		// sets can be have a byte size between 2 & 4 bytes.
		index += skip
	}

	for index < len(data) {
		// If the sequence satisfies the sentence termination conditions, return
		// the values upon which to termination point in a sentence
		if shouldBreakSentenceAtMiddleChar(seq) {
			return index - skip, data[:index-skip], nil
		}

		// Else check next triple array sequence.
		seq[0], seq[1] = seq[1], seq[2]
		if seq[2], skip = scanRune(data[index:]); skip < 0 {
			return 0, nil, nil
		}

		index += skip
	}

	return 0, nil, nil
}

// shouldBreakSentenceAtMiddleChar runs checks on three rune / byte sequences
// to determine if they contain termination symbols such as ['.', '!', '?'].
func shouldBreakSentenceAtMiddleChar(seq [3]rune) bool {
	condition1 := (unicode.IsLower(seq[0]) || unicode.IsSymbol(seq[0]) ||
		unicode.IsNumber(seq[0]) || unicode.IsSpace(seq[0]))

	condition2 := (seq[1] == '.' || seq[1] == '!' || seq[1] == '?')

	condition3 := (unicode.IsPunct(seq[2]) || unicode.IsSpace(seq[2]) ||
		unicode.IsSymbol(seq[2]) || unicode.IsNumber(seq[2]) ||
		unicode.IsUpper(seq[2]))

	return condition1 && condition2 && condition3
}

func scanRune(data []byte) (rune, int) {
	if len(data) == 0 {
		return 0, -1
	}

	// Check for ASCII char.
	if data[0] < utf8.RuneSelf {
		return rune(data[0]), 1
	}

	// Attempt to successfully decode non ASCII byte data into a valid
	// UTF-8 rune. Returning a size > 1 means the provided byte represents
	// a character binary outside the original ASCII character range. ie emoji's.
	r, size := utf8.DecodeRune(data)
	if size > 1 {
		return r, size
	}

	// Byte at data[0] position doesn't belong to the unicode character set.
	return 0, -1
}
