package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Word struct {
	German   string   `json:"german"`
	English  string   `json:"english"`
	Level    string   `json:"level"`
	Examples []string `json:"examples"`
	Synonyms []string `json:"synonyms"`
	Antonyms []string `json:"antonyms"`
}

type WiseSentences struct {
	German  string `json:"german"`
	English string `json:"english"`
}

type UserProgress struct {
	ChatID       string   `json:"chat_id"`
	LearnedWords []string `json:"learned_words"`
	LastStudy    string   `json:"last_study_date"`
}

const chatIDFile = "chat_ids.json"
const userProgressDir = "user_progress"

func main() {
	fmt.Println("Starting Daily German Study Bot...")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	// 1. 명령어 처리 (/learned, /learn)
	processCommands(botToken)

	// 2. /start 누른 사용자 새로 불러오기
	newIDs := fetchNewChatIDs(botToken)
	mergeChatIDs(newIDs)

	// 3. 모든 사용자에게 맞춤형 단어 전송
	chatIDs := loadChatIDs()
	sentence := selectDailySentence()

	for _, id := range chatIDs {
		words := selectDailyWordsForUser(id)
		message := formatMessage(words, sentence)
		sendToTelegram(botToken, id, message)
	}
}

// ---------------- 명령어 처리 ----------------
func processCommands(botToken string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", botToken)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error fetching updates:", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Result []struct {
			Message struct {
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	for _, update := range result.Result {
		chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
		text := strings.TrimSpace(update.Message.Text)

		if strings.HasPrefix(text, "/learned ") {
			handleLearnedCommand(botToken, chatID, text)
		} else if strings.HasPrefix(text, "/learn ") {
			handleLearnLevelCommand(botToken, chatID, text)
		} else if text == "/stats" {
			handleStatsCommand(botToken, chatID)
		}
	}
}

func handleLearnedCommand(botToken, chatID, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		sendToTelegram(botToken, chatID, "📝 사용법: /learned Hallo Tschüss Danke")
		return
	}

	words := parts[1:] // /learned 제외한 나머지
	progress := loadUserProgress(chatID)

	// 중복 제거하며 추가
	learnedMap := make(map[string]bool)
	for _, w := range progress.LearnedWords {
		learnedMap[w] = true
	}

	newCount := 0
	for _, w := range words {
		if !learnedMap[w] {
			progress.LearnedWords = append(progress.LearnedWords, w)
			learnedMap[w] = true
			newCount++
		}
	}

	progress.LastStudy = time.Now().Format("2006-01-02")
	saveUserProgress(progress)

	msg := fmt.Sprintf("✅ *%d개 단어*를 학습 완료로 기록했어요!\n📚 총 학습: *%d개*",
		newCount, len(progress.LearnedWords))
	sendToTelegram(botToken, chatID, msg)
}

func handleLearnLevelCommand(botToken, chatID, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		sendToTelegram(botToken, chatID, "📝 사용법: /learn a1, /learn a2, /learn b1")
		return
	}

	level := strings.ToLower(parts[1])
	var filename string

	switch level {
	case "a1":
		filename = "vocabulary/a1_words.json"
	case "a2":
		filename = "vocabulary/a2_words.json"
	case "b1":
		filename = "vocabulary/b1_words.json"
	default:
		sendToTelegram(botToken, chatID, "❌ 지원하는 레벨: a1, a2, b1")
		return
	}

	// 해당 레벨 단어 로드
	data, err := os.ReadFile(filename)
	if err != nil {
		sendToTelegram(botToken, chatID, "⚠️ 단어 파일을 찾을 수 없습니다.")
		return
	}

	var allWords []Word
	if err := json.Unmarshal(data, &allWords); err != nil {
		sendToTelegram(botToken, chatID, "⚠️ 파일 파싱 오류")
		return
	}

	// 유저 진행도 로드
	progress := loadUserProgress(chatID)
	learnedMap := make(map[string]bool)
	for _, w := range progress.LearnedWords {
		learnedMap[w] = true
	}

	// 안 배운 단어만 필터링
	var unlearned []Word
	for _, word := range allWords {
		if !learnedMap[word.German] {
			unlearned = append(unlearned, word)
		}
	}

	if len(unlearned) == 0 {
		msg := fmt.Sprintf("🎉 *%s 레벨 완료!*\n\n모든 단어를 학습했어요!", strings.ToUpper(level))
		sendToTelegram(botToken, chatID, msg)
		return
	}

	// 랜덤 셔플
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(unlearned), func(i, j int) {
		unlearned[i], unlearned[j] = unlearned[j], unlearned[i]
	})

	// 최대 10개 선택
	count := 10
	if len(unlearned) < count {
		count = len(unlearned)
	}
	selectedWords := unlearned[:count]

	// 메시지 포맷
	sentence := selectDailySentence()
	message := formatLevelMessage(selectedWords, sentence, level)
	sendToTelegram(botToken, chatID, message)
}

func formatLevelMessage(words []Word, sentence WiseSentences, level string) string {
	msg := fmt.Sprintf("🇩🇪 *%s Level Study* 🇩🇪\n\n", strings.ToUpper(level))

	for i, word := range words {
		msg += fmt.Sprintf("*%d. %s*\n", i+1, word.German)
		msg += fmt.Sprintf("📖 %s\n\n", word.English)
		for _, ex := range word.Examples {
			msg += fmt.Sprintf("💬 %s\n\n", ex)
		}
		if len(word.Synonyms) > 0 {
			msg += fmt.Sprintf("🔄 Synonyms: %v\n\n", word.Synonyms)
		}
		if len(word.Antonyms) > 0 {
			msg += fmt.Sprintf("🔀 Antonyms: %v\n\n", word.Antonyms)
		}
		msg += "\n---\n\n"
	}

	msg += "💡 *Wise Sentence*\n\n"
	msg += fmt.Sprintf("🇩🇪 %s\n", sentence.German)
	msg += fmt.Sprintf("🇬🇧 %s\n\n", sentence.English)
	msg += "_/learned [words] to mark as learned_"

	return msg
}

func handleStatsCommand(botToken, chatID string) {
	progress := loadUserProgress(chatID)

	// 레벨별 통계 계산
	a1Total := len(loadWordsByLevel("vocabulary/a1_words.json"))
	a2Total := len(loadWordsByLevel("vocabulary/a2_words.json"))
	b1Total := len(loadWordsByLevel("vocabulary/b1_words.json"))
	totalWords := a1Total + a2Total + b1Total

	learned := len(progress.LearnedWords)
	remaining := totalWords - learned
	percentage := 0
	if totalWords > 0 {
		percentage = (learned * 100) / totalWords
	}

	msg := fmt.Sprintf("📊 *학습 통계*\n\n"+
		"✅ 학습 완료: *%d개*\n"+
		"📝 남은 단어: *%d개*\n"+
		"📈 진행도: *%d%%*\n\n"+
		"📚 전체 단어: %d개\n"+
		"   • A1: %d개\n"+
		"   • A2: %d개\n"+
		"   • B1: %d개\n\n"+
		"📅 마지막 학습: %s",
		learned, remaining, percentage,
		totalWords, a1Total, a2Total, b1Total,
		progress.LastStudy)

	sendToTelegram(botToken, chatID, msg)
}

func loadWordsByLevel(filename string) []string {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", filename, err)
		return []string{}
	}

	var words []Word
	if err := json.Unmarshal(data, &words); err != nil {
		fmt.Printf("Error parsing %s: %v\n", filename, err)
		return []string{}
	}

	result := make([]string, len(words))
	for i, w := range words {
		result[i] = w.German
	}
	return result
}

// ---------------- 유저 진행도 관리 ----------------
func loadUserProgress(chatID string) UserProgress {
	progressFile := filepath.Join(userProgressDir, chatID+"_progress.json")

	if data, err := os.ReadFile(progressFile); err == nil {
		var progress UserProgress
		if err := json.Unmarshal(data, &progress); err == nil {
			return progress
		}
	}

	// 파일이 없으면 새로 생성
	return UserProgress{
		ChatID:       chatID,
		LearnedWords: []string{},
		LastStudy:    "처음",
	}
}

func saveUserProgress(progress UserProgress) {
	os.MkdirAll(userProgressDir, 0755)
	progressFile := filepath.Join(userProgressDir, progress.ChatID+"_progress.json")

	data, _ := json.MarshalIndent(progress, "", "  ")
	if err := os.WriteFile(progressFile, data, 0644); err != nil {
		fmt.Printf("Error saving progress: %v\n", err)
	} else {
		fmt.Printf("✓ Saved progress for user %s (learned: %d)\n", progress.ChatID, len(progress.LearnedWords))
	}
}

// ---------------- getUpdates로 /start 감지 ----------------
func fetchNewChatIDs(botToken string) []string {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", botToken)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error fetching new chat IDs:", err)
		return []string{}
	}
	defer resp.Body.Close()

	var result struct {
		Result []struct {
			Message struct {
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	var newIDs []string
	for _, update := range result.Result {
		if update.Message.Text == "/start" {
			newIDs = append(newIDs, fmt.Sprintf("%d", update.Message.Chat.ID))
		}
	}

	if len(newIDs) > 0 {
		fmt.Printf("Fetched %d new chat IDs from /start commands.\n", len(newIDs))
	}

	return newIDs
}

// ---------------- chat_ids.json 관리 ----------------
func loadChatIDs() []string {
	if _, err := os.Stat(chatIDFile); os.IsNotExist(err) {
		return []string{}
	}
	data, _ := os.ReadFile(chatIDFile)
	var ids []string
	json.Unmarshal(data, &ids)
	return ids
}

func mergeChatIDs(newIDs []string) {
	ids := loadChatIDs()
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	for _, id := range newIDs {
		if !idMap[id] {
			ids = append(ids, id)
		}
	}

	data, _ := json.Marshal(ids)
	os.WriteFile(chatIDFile, data, 0644)
	fmt.Println("chat_ids.json updated locally")
}

// ---------------- 단어 선택 (유저별 맞춤) ----------------
func selectDailyWordsForUser(chatID string) []Word {
	// 전체 단어 로드
	a1File, _ := os.ReadFile("vocabulary/a1_words.json")
	a2File, _ := os.ReadFile("vocabulary/a2_words.json")
	b1File, _ := os.ReadFile("vocabulary/b1_words.json")

	var a1Words, a2Words, b1Words []Word
	json.Unmarshal(a1File, &a1Words)
	json.Unmarshal(a2File, &a2Words)
	json.Unmarshal(b1File, &b1Words)

	allWords := append(append(a1Words, a2Words...), b1Words...)

	// 유저가 배운 단어 로드
	progress := loadUserProgress(chatID)
	learnedMap := make(map[string]bool)
	for _, word := range progress.LearnedWords {
		learnedMap[word] = true
	}

	// 안 배운 단어만 필터링
	var unlearned []Word
	for _, word := range allWords {
		if !learnedMap[word.German] {
			unlearned = append(unlearned, word)
		}
	}

	fmt.Printf("User %s: %d learned, %d unlearned words\n",
		chatID, len(progress.LearnedWords), len(unlearned))

	// 단어가 부족하면 있는 만큼만 반환
	if len(unlearned) == 0 {
		return []Word{} // 모든 단어 학습 완료
	}

	// 레벨별로 분류
	var a1Unlearned, a2Unlearned, b1Unlearned []Word
	for _, word := range unlearned {
		switch word.Level {
		case "A1":
			a1Unlearned = append(a1Unlearned, word)
		case "A2":
			a2Unlearned = append(a2Unlearned, word)
		case "B1":
			b1Unlearned = append(b1Unlearned, word)
		}
	}

	// 각 레벨별로 셔플
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(a1Unlearned), func(i, j int) {
		a1Unlearned[i], a1Unlearned[j] = a1Unlearned[j], a1Unlearned[i]
	})
	rand.Shuffle(len(a2Unlearned), func(i, j int) {
		a2Unlearned[i], a2Unlearned[j] = a2Unlearned[j], a2Unlearned[i]
	})
	rand.Shuffle(len(b1Unlearned), func(i, j int) {
		b1Unlearned[i], b1Unlearned[j] = b1Unlearned[j], b1Unlearned[i]
	})

	// A1 3개, A2 3개, B1 4개 선택 (가능한 범위 내에서)
	var selected []Word
	selected = append(selected, takeWords(a1Unlearned, 3)...)
	selected = append(selected, takeWords(a2Unlearned, 3)...)
	selected = append(selected, takeWords(b1Unlearned, 4)...)

	return selected
}

func takeWords(words []Word, count int) []Word {
	if len(words) <= count {
		return words
	}
	return words[:count]
}

// ---------------- 명언 선택 ----------------
func selectDailySentence() WiseSentences {
	file, _ := os.ReadFile("vocabulary/sentences.json")
	var sentences []WiseSentences
	json.Unmarshal(file, &sentences)
	rand.Seed(time.Now().UnixNano())
	return sentences[rand.Intn(len(sentences))]
}

// ---------------- 메시지 포맷 ----------------
func formatMessage(words []Word, sentence WiseSentences) string {
	if len(words) == 0 {
		return "🎉 *축하합니다!*\n\n모든 단어를 학습하셨네요!\n\n💪 대단해요!"
	}

	msg := `
Tip: /learned [words] to mark learned
/learn a1/a2/b1 to learn level specific words
/stats for progress

🇩🇪 *Today's German Study* 🇩🇪
`

	for i, word := range words {
		msg += fmt.Sprintf("(%s) *%d. %s*\n", word.Level, i+1, word.German)
		msg += fmt.Sprintf("📖 %s\n\n", word.English)
		for _, ex := range word.Examples {
			msg += fmt.Sprintf("💬 %s\n\n", ex)
		}
		if len(word.Synonyms) > 0 {
			msg += fmt.Sprintf("🔄 Synonyms: %v\n\n", word.Synonyms)
		}
		if len(word.Antonyms) > 0 {
			msg += fmt.Sprintf("🔀 Antonyms: %v\n\n", word.Antonyms)
		}
		msg += "\n---\n\n"
	}
	msg += "💡 *Wise Sentence of the Day*\n\n"
	msg += fmt.Sprintf("🇩🇪 %s\n", sentence.German)
	msg += fmt.Sprintf("🇬🇧 %s\n\n", sentence.English)
	return msg
}

// ---------------- 텔레그램 전송 ----------------
func sendToTelegram(botToken, chatID, message string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)
	data.Set("parse_mode", "Markdown")

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		fmt.Printf("Error sending message to %s: %v\n", chatID, err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✓ Sent message to %s\n", chatID)
}
