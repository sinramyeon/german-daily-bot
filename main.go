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

type LevelProgress struct {
	A1 []string `json:"a1"`
	A2 []string `json:"a2"`
	B1 []string `json:"b1"`
}

type UserProgress struct {
	ChatID          string        `json:"chat_id"`
	LearnedWords    LevelProgress `json:"learned_words"`
	LastStudy       string        `json:"last_study_date"`
	LastUpdateID    int           `json:"last_update_id"`
	WelcomeSent     bool          `json:"welcome_sent"`
	LastWelcomeDate string        `json:"last_welcome_date"`
}

const chatIDFile = "chat_ids.json"
const userProgressDir = "user_progress"

func main() {
	fmt.Println("Starting German Study Bot - Command Processor...")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	if botToken == "" {
		fmt.Println("Error: TELEGRAM_BOT_TOKEN not set")
		return
	}

	// 월요일 8am인지 확인하고 환영 메시지 전송
	sendMondayWelcomeIfNeeded(botToken)

	// 명령어 처리 (/start, /learn, /learned, /stats)
	processCommands(botToken)
}

// ---------------- 월요일 환영 메시지 ----------------
func sendMondayWelcomeIfNeeded(botToken string) {
	now := time.Now()

	// 월요일이고 시간이 8am인지 확인
	if now.Weekday() != time.Monday || now.Hour() != 8 {
		return
	}

	chatIDs := loadChatIDs()
	today := now.Format("2006-01-02")

	welcomeMsg := `🇩🇪 *Weekly German Study Guide* 🇩🇪

안녕하세요! 이번 주도 독일어 공부를 시작해볼까요? 😊

*📚 사용 가능한 명령어:*

*1. /learn [level]*
   특정 레벨의 단어 10개를 학습합니다
   예: /learn a1, /learn a2, /learn b1

*2. /learned [단어들]*
   학습 완료한 단어를 기록합니다
   예: /learned Hallo Tschüss Danke

*3. /stats*
   현재 학습 진행 상황을 확인합니다

*💡 추천 학습 방법:*
• 매일 /learn 명령어로 새 단어 학습
• 익힌 단어는 /learned로 기록
• 주기적으로 /stats로 진행도 확인

화이팅! 💪`

	for _, chatID := range chatIDs {
		progress := loadUserProgress(chatID)

		// 오늘 이미 환영 메시지를 보냈는지 확인
		if progress.LastWelcomeDate == today {
			continue
		}

		sendToTelegram(botToken, chatID, welcomeMsg)

		// 환영 메시지 전송 기록
		progress.LastWelcomeDate = today
		saveUserProgress(progress)

		time.Sleep(100 * time.Millisecond) // Rate limiting
	}
}

// ---------------- 명령어 처리 ----------------
func processCommands(botToken string) {
	chatIDs := loadChatIDs()

	// 모든 사용자의 새 메시지 확인
	for _, chatID := range chatIDs {
		processUserCommands(botToken, chatID)
	}

	// /start로 새로 등록된 사용자 확인
	checkNewUsers(botToken)
}

func processUserCommands(botToken, chatID string) {
	progress := loadUserProgress(chatID)

	// getUpdates with offset
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&allowed_updates=[\"message\"]",
		botToken, progress.LastUpdateID+1)

	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Printf("Error fetching updates for %s: %v\n", chatID, err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Ok     bool `json:"ok"`
		Result []struct {
			UpdateID int `json:"update_id"`
			Message  struct {
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error decoding response for %s: %v\n", chatID, err)
		return
	}

	if !result.Ok || len(result.Result) == 0 {
		return
	}

	// 이 사용자의 메시지만 처리
	for _, update := range result.Result {
		if fmt.Sprintf("%d", update.Message.Chat.ID) != chatID {
			continue
		}

		text := strings.TrimSpace(update.Message.Text)

		if strings.HasPrefix(text, "/learn ") {
			handleLearnLevelCommand(botToken, chatID, text)
		} else if strings.HasPrefix(text, "/learned ") {
			handleLearnedCommand(botToken, chatID, text)
		} else if text == "/stats" {
			handleStatsCommand(botToken, chatID)
		}

		// Update ID 갱신
		if update.UpdateID > progress.LastUpdateID {
			progress.LastUpdateID = update.UpdateID
		}
	}

	// 진행도 저장
	if len(result.Result) > 0 {
		saveUserProgress(progress)
	}
}

func checkNewUsers(botToken string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", botToken)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error checking new users:", err)
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

	newUsers := []string{}
	for _, update := range result.Result {
		if update.Message.Text == "/start" {
			chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
			if !isChatIDRegistered(chatID) {
				newUsers = append(newUsers, chatID)

				// 환영 메시지 전송
				welcomeMsg := `🇩🇪 *German Study Bot에 오신 것을 환영합니다!* 🇩🇪

안녕하세요! 독일어 학습을 도와드리겠습니다. 😊

*📚 사용 가능한 명령어:*

*1. /learn [level]*
   특정 레벨의 단어 10개를 학습합니다
   • /learn a1 - 기초 단어
   • /learn a2 - 초급 단어
   • /learn b1 - 중급 단어

*2. /learned [단어들]*
   학습 완료한 단어를 기록합니다
   예: /learned Hallo Tschüss Danke

*3. /stats*
   현재 학습 진행 상황을 확인합니다

*💡 시작하기:*
/learn a1 명령어로 첫 단어를 배워보세요!

매주 월요일 아침 8시에 학습 가이드를 보내드립니다.`

				sendToTelegram(botToken, chatID, welcomeMsg)
			}
		}
	}

	if len(newUsers) > 0 {
		mergeChatIDs(newUsers)
		fmt.Printf("Added %d new users\n", len(newUsers))
	}
}

func isChatIDRegistered(chatID string) bool {
	ids := loadChatIDs()
	for _, id := range ids {
		if id == chatID {
			return true
		}
	}
	return false
}

func handleLearnedCommand(botToken, chatID, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		sendToTelegram(botToken, chatID, "📝 *사용법*\n\n/learned Hallo Tschüss Danke\n\n학습한 단어들을 띄어쓰기로 구분해서 입력하세요.")
		return
	}

	words := parts[1:] // /learned 제외한 나머지
	progress := loadUserProgress(chatID)

	// 단어를 레벨별로 분류하여 저장
	levelMap := buildLevelMap()

	newWordsA1 := []string{}
	newWordsA2 := []string{}
	newWordsB1 := []string{}
	unknownWords := []string{}

	// 각 레벨별 중복 체크용 맵 생성
	a1Map := make(map[string]bool)
	a2Map := make(map[string]bool)
	b1Map := make(map[string]bool)

	for _, w := range progress.LearnedWords.A1 {
		a1Map[w] = true
	}
	for _, w := range progress.LearnedWords.A2 {
		a2Map[w] = true
	}
	for _, w := range progress.LearnedWords.B1 {
		b1Map[w] = true
	}

	// 입력된 단어를 레벨별로 분류하고 중복 체크
	for _, word := range words {
		level, exists := levelMap[word]
		if !exists {
			unknownWords = append(unknownWords, word)
			continue
		}

		switch level {
		case "A1":
			if !a1Map[word] {
				progress.LearnedWords.A1 = append(progress.LearnedWords.A1, word)
				a1Map[word] = true
				newWordsA1 = append(newWordsA1, word)
			}
		case "A2":
			if !a2Map[word] {
				progress.LearnedWords.A2 = append(progress.LearnedWords.A2, word)
				a2Map[word] = true
				newWordsA2 = append(newWordsA2, word)
			}
		case "B1":
			if !b1Map[word] {
				progress.LearnedWords.B1 = append(progress.LearnedWords.B1, word)
				b1Map[word] = true
				newWordsB1 = append(newWordsB1, word)
			}
		}
	}

	progress.LastStudy = time.Now().Format("2006-01-02")
	saveUserProgress(progress)

	// 응답 메시지 생성
	totalNew := len(newWordsA1) + len(newWordsA2) + len(newWordsB1)
	totalLearned := len(progress.LearnedWords.A1) + len(progress.LearnedWords.A2) + len(progress.LearnedWords.B1)

	msg := fmt.Sprintf("✅ *%d개 단어*를 학습 완료로 기록했어요!\n\n", totalNew)

	if len(newWordsA1) > 0 {
		msg += fmt.Sprintf("🟢 *A1:* %s\n", strings.Join(newWordsA1, ", "))
	}
	if len(newWordsA2) > 0 {
		msg += fmt.Sprintf("🟡 *A2:* %s\n", strings.Join(newWordsA2, ", "))
	}
	if len(newWordsB1) > 0 {
		msg += fmt.Sprintf("🔵 *B1:* %s\n", strings.Join(newWordsB1, ", "))
	}

	if len(unknownWords) > 0 {
		msg += fmt.Sprintf("\n⚠️ *미등록 단어:* %s\n", strings.Join(unknownWords, ", "))
	}

	msg += fmt.Sprintf("\n📚 *총 학습 완료:* %d개\n\n", totalLearned)
	msg += "계속 화이팅! 💪"

	sendToTelegram(botToken, chatID, msg)
}

func handleLearnLevelCommand(botToken, chatID, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		sendToTelegram(botToken, chatID, "📝 *사용법*\n\n/learn a1\n/learn a2\n/learn b1\n\n레벨을 선택하세요!")
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
		sendToTelegram(botToken, chatID, "❌ *지원하는 레벨*\n\na1, a2, b1")
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

	// 해당 레벨의 학습 완료 단어만 맵으로 변환
	learnedMap := make(map[string]bool)
	var learnedList []string

	switch level {
	case "a1":
		learnedList = progress.LearnedWords.A1
	case "a2":
		learnedList = progress.LearnedWords.A2
	case "b1":
		learnedList = progress.LearnedWords.B1
	}

	for _, w := range learnedList {
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
		msg := fmt.Sprintf("🎉 *%s 레벨 완료!*\n\n모든 단어를 학습했어요!\n\n", strings.ToUpper(level))
		msg += "다른 레벨도 도전해보세요! 💪"
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
			msg += fmt.Sprintf("🔄 Synonyms: %s\n\n", strings.Join(word.Synonyms, ", "))
		}
		if len(word.Antonyms) > 0 {
			msg += fmt.Sprintf("🔀 Antonyms: %s\n\n", strings.Join(word.Antonyms, ", "))
		}
		msg += "---\n\n"
	}

	msg += "💡 *Wise Sentence*\n\n"
	msg += fmt.Sprintf("🇩🇪 %s\n", sentence.German)
	msg += fmt.Sprintf("🇬🇧 %s\n\n", sentence.English)
	msg += "_학습한 단어는 /learned [단어들]로 기록하세요_"

	return msg
}

func handleStatsCommand(botToken, chatID string) {
	progress := loadUserProgress(chatID)

	// 레벨별 통계 계산
	a1Total := len(loadWordsByLevel("vocabulary/a1_words.json"))
	a2Total := len(loadWordsByLevel("vocabulary/a2_words.json"))
	b1Total := len(loadWordsByLevel("vocabulary/b1_words.json"))
	totalWords := a1Total + a2Total + b1Total

	a1Learned := len(progress.LearnedWords.A1)
	a2Learned := len(progress.LearnedWords.A2)
	b1Learned := len(progress.LearnedWords.B1)
	learned := a1Learned + a2Learned + b1Learned

	remaining := totalWords - learned
	percentage := 0
	if totalWords > 0 {
		percentage = (learned * 100) / totalWords
	}

	msg := fmt.Sprintf("📊 *학습 통계*\n\n"+
		"✅ *학습 완료:* %d개\n"+
		"📝 *남은 단어:* %d개\n"+
		"📈 *진행도:* %d%%\n\n"+
		"---\n\n"+
		"📚 *레벨별 진행도*\n\n"+
		"🟢 A1: %d/%d (%d%%)\n"+
		"🟡 A2: %d/%d (%d%%)\n"+
		"🔵 B1: %d/%d (%d%%)\n\n"+
		"---\n\n"+
		"📅 *마지막 학습:* %s\n\n"+
		"계속 화이팅! 💪",
		learned, remaining, percentage,
		a1Learned, a1Total, getPercentage(a1Learned, a1Total),
		a2Learned, a2Total, getPercentage(a2Learned, a2Total),
		b1Learned, b1Total, getPercentage(b1Learned, b1Total),
		progress.LastStudy)

	sendToTelegram(botToken, chatID, msg)
}

func getPercentage(learned, total int) int {
	if total == 0 {
		return 0
	}
	return (learned * 100) / total
}

func loadWordsByLevel(filename string) []string {
	data, err := os.ReadFile(filename)
	if err != nil {
		return []string{}
	}

	var words []Word
	if err := json.Unmarshal(data, &words); err != nil {
		return []string{}
	}

	result := make([]string, len(words))
	for i, w := range words {
		result[i] = w.German
	}
	return result
}

// 모든 단어의 레벨 맵 생성 (단어 -> 레벨)
func buildLevelMap() map[string]string {
	levelMap := make(map[string]string)

	// A1
	a1Data, _ := os.ReadFile("vocabulary/a1_words.json")
	var a1Words []Word
	json.Unmarshal(a1Data, &a1Words)
	for _, w := range a1Words {
		levelMap[w.German] = "A1"
	}

	// A2
	a2Data, _ := os.ReadFile("vocabulary/a2_words.json")
	var a2Words []Word
	json.Unmarshal(a2Data, &a2Words)
	for _, w := range a2Words {
		levelMap[w.German] = "A2"
	}

	// B1
	b1Data, _ := os.ReadFile("vocabulary/b1_words.json")
	var b1Words []Word
	json.Unmarshal(b1Data, &b1Words)
	for _, w := range b1Words {
		levelMap[w.German] = "B1"
	}

	return levelMap
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
		ChatID: chatID,
		LearnedWords: LevelProgress{
			A1: []string{},
			A2: []string{},
			B1: []string{},
		},
		LastStudy:    "처음",
		LastUpdateID: 0,
	}
}

func saveUserProgress(progress UserProgress) {
	os.MkdirAll(userProgressDir, 0755)
	progressFile := filepath.Join(userProgressDir, progress.ChatID+"_progress.json")

	data, _ := json.MarshalIndent(progress, "", "  ")
	os.WriteFile(progressFile, data, 0644)
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
}

// ---------------- 명언 선택 ----------------
func selectDailySentence() WiseSentences {
	file, _ := os.ReadFile("vocabulary/sentences.json")
	var sentences []WiseSentences
	json.Unmarshal(file, &sentences)
	rand.Seed(time.Now().UnixNano())
	return sentences[rand.Intn(len(sentences))]
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
