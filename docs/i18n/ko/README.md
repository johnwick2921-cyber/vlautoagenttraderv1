names rewritten to vl on 2026-09-30 (VL rename)
<h1 align="center">VL Intelligent</h1>

<p align="center">
  <strong>당신만의 AI 트레이딩 어시스턴트.</strong><br/>

</p>

<p align="center">
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go"></a>
  <a href="https://reactjs.org/"><img src="https://img.shields.io/badge/React-18+-61DAFB?style=flat&logo=react" alt="React"></a>
</p>

<p align="center">
  <a href="../../../README.md">English</a> ·
  <a href="../zh-CN/README.md">中文</a> ·
  <a href="../ja/README.md">日本語</a> ·
  <a href="README.md">한국어</a> ·
  <a href="../ru/README.md">Русский</a> ·
  <a href="../uk/README.md">Українська</a> ·
  <a href="../vi/README.md">Tiếng Việt</a>
</p>

---

VL Intelligent는 오픈소스 **자율형** AI 트레이딩 어시스턴트입니다. 수동으로 모델을 설정하고, API 키를 관리하고, 데이터 소스를 연결해야 하는 기존 AI 도구와 달리 — VL Intelligent의 AI는 **시장을 스스로 인식하고, 모델을 스스로 선택하고, 데이터를 스스로 가져옵니다**. 인간 개입 제로. 전략만 설정하면 나머지는 AI가 처리합니다.





**http://127.0.0.1:3000** 을 열면 완료.

---

## 빠른 데모

<p align="center">
  <a href="https://drive.google.com/file/d/1frzw-HDZ3viQvLOQKsAJGc9bT0dXs68D/view">
    <img src="../../../screenshots/demo-cover.png" alt="VL Intelligent 빠른 데모 영상" width="900"/>
  </a>
</p>

<p align="center">
  커버 이미지를 클릭하면 데모 영상을 볼 수 있습니다.
</p>

---

## 기능

| 기능 | 설명 |
| **멀티 AI** | DeepSeek, Qwen, GPT, Claude, Gemini, Grok, Kimi, MiniMax — 언제든 전환 |

| **전략 스튜디오** | 비주얼 빌더 — 코인 소스, 지표, 리스크 관리 |
| **AI 토론 아레나** | 여러 AI가 거래 토론 (강세 vs 약세 vs 분석가), 투표, 실행 |
| **AI 경쟁** | AI가 실시간 경쟁, 리더보드 순위 |
| **Telegram 에이전트** | 트레이딩 어시스턴트와 채팅 — 스트리밍, 도구 호출, 메모리 |
| **백테스트 랩** | 과거 시뮬레이션, 자산 곡선 및 성과 지표 |
| **대시보드** | 실시간 포지션, 손익, Chain of Thought AI 결정 로그 |

### 시장

암호화폐 · 미국 주식 · 외환 · 귀금속

### AI 모델 (API 키 모드)

| AI 모델 | 상태 | API 키 받기 |
| <img src="../../../web/public/icons/deepseek.svg" width="20" height="20" style="vertical-align: middle;"/> **DeepSeek** | ✅ | [API 키 받기](https://platform.deepseek.com) |
| <img src="../../../web/public/icons/qwen.svg" width="20" height="20" style="vertical-align: middle;"/> **Qwen** | ✅ | [API 키 받기](https://dashscope.console.aliyun.com) |
| <img src="../../../web/public/icons/openai.svg" width="20" height="20" style="vertical-align: middle;"/> **OpenAI (GPT)** | ✅ | [API 키 받기](https://platform.openai.com) |
| <img src="../../../web/public/icons/claude.svg" width="20" height="20" style="vertical-align: middle;"/> **Claude** | ✅ | [API 키 받기](https://console.anthropic.com) |
| <img src="../../../web/public/icons/gemini.svg" width="20" height="20" style="vertical-align: middle;"/> **Gemini** | ✅ | [API 키 받기](https://aistudio.google.com) |
| <img src="../../../web/public/icons/grok.svg" width="20" height="20" style="vertical-align: middle;"/> **Grok** | ✅ | [API 키 받기](https://console.x.ai) |
| <img src="../../../web/public/icons/kimi.svg" width="20" height="20" style="vertical-align: middle;"/> **Kimi** | ✅ | [API 키 받기](https://platform.moonshot.cn) |
| <img src="../../../web/public/icons/minimax.svg" width="20" height="20" style="vertical-align: middle;"/> **MiniMax** | ✅ | [API 키 받기](https://platform.minimaxi.com) |

---

## 설치

### 소스에서

```bash
# 필수 조건: Go 1.21+, Node.js 18+, TA-Lib
# macOS: brew install ta-lib

go build -o vl-bin && ./vl-bin          # 백엔드
cd web && npm install && npm run dev  # 프론트엔드 (새 터미널)
```

---

## 링크

> **위험 경고**: AI 자동 거래에는 상당한 위험이 있습니다. 학습/연구 또는 소액 테스트만 권장합니다.

---

## License

[AGPL-3.0](../../../LICENSE)

