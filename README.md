# 🇩🇪 Daily German Study Bot

GitHub Actions로 서버 없이 운영하는 독일어 학습 텔레그램 봇

## ✨ 기능

### 📚 자동 단어 전송
- 매일 8am, 8pm에 자동으로 독일어 단어 전송
- A1 3개, A2 3개, B1 4개 = 총 10개 단어
- 유저별 **학습한 단어 제외**하고 전송

### 🎯 개인화 학습 관리
- `/learned Hallo Tschüss` - 개별 단어 학습 완료 기록
- `/learn a1` - A1 레벨에서 10개 단어 즉시 학습
- `/learn a2` - A2 레벨에서 10개 단어 즉시 학습
- `/learn b1` - B1 레벨에서 10개 단어 즉시 학습
- `/stats` - 학습 진행도 확인

### 💡 추가 기능
- 매일 랜덤 명언 전송
- 예문, 동의어, 반의어 포함

## 🚀 사용법

### 1. 봇 시작
```
/start
```

### 2. 단어 학습 완료 표시
```
/learned Hallo Tschüss Danke Bitte
```
→ 해당 단어들이 다시는 안 나옴

### 3. 레벨별 단어 배우기
```
/learn a1
```
→ A1 레벨에서 10개 단어 즉시 출력

```
/learn a2
/learn b1
```
→ 각 레벨별로 10개씩 학습 가능

### 4. 진행도 확인
```
/stats
```

출력 예시:
```
📊 학습 통계

✅ 학습 완료: 150개
📝 남은 단어: 2854개
📈 진행도: 5%

📚 전체 단어: 3004개
   • A1: 1020개
   • A2: 1006개
   • B1: 978개

📅 마지막 학습: 2024-12-10
```

## 📁 프로젝트 구조

```
.
├── main.go
├── vocabulary/
│   ├── a1_words.json
│   ├── a2_words.json
│   ├── b1_words.json
│   └── sentences.json
├── chat_ids.json              # 자동 생성됨
└── user_progress/             # 자동 생성됨
    ├── 123456_progress.json
    └── 789012_progress.json
```

## 🔧 설정

### GitHub Secrets
`TELEGRAM_BOT_TOKEN` 설정 필요

### 크론 일정
```yaml
- cron: "0 7 * * *"   # 08:00 (Berlin time)
- cron: "0 19 * * *"  # 20:00 (Berlin time)
```

## 📊 데이터 구조

### user_progress/{chat_id}_progress.json
```json
{
  "chat_id": "123456789",
  "learned_words": ["Hallo", "Tschüss", "Danke"],
  "last_study_date": "2024-12-10"
}
```

## 🎯 알고리즘

1. **명령어 처리**: `/learned`, `/learn` 먼저 처리
2. **유저별 필터링**: learned_words에 있는 단어 제외
3. **레벨별 선택**: A1(3), A2(3), B1(4) 비율 유지
4. **랜덤 셔플**: 매일 다른 순서로 전송

## 💰 비용

**완전 무료!**
- GitHub Actions: 월 2,000분 무료
- 하루 2번 실행 × 1분 = 월 60분 사용
- 텔레그램 Bot API: 무료

## 🔮 향후 계획

- [ ] B2 레벨 추가
- [ ] 비즈니스/IT/건강 등 주제별 단어
- [ ] 퀴즈 모드
- [ ] Spaced Repetition 알고리즘
- [ ] 주간 복습 단어

## 📝 라이센스

MIT