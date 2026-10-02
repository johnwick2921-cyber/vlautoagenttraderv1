names rewritten to vl on 2026-09-30 (VL rename)
<h1 align="center">VL Intelligent</h1>

<p align="center">
  <strong>Ваш персональный AI торговый ассистент.</strong><br/>

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
  <a href="../ko/README.md">한국어</a> ·
  <a href="README.md">Русский</a> ·
  <a href="../uk/README.md">Українська</a> ·
  <a href="../vi/README.md">Tiếng Việt</a>
</p>

---

VL Intelligent — это **автономный** AI торговый ассистент с открытым исходным кодом. В отличие от традиционных AI инструментов, где нужно вручную настраивать модели, управлять API ключами и подключать источники данных — AI в VL Intelligent **сам анализирует рынки, выбирает модели и получает данные**. Нулевое вмешательство человека. Вы задаёте стратегию, AI делает всё остальное.





Откройте **http://127.0.0.1:3000**. Готово.

---

## Быстрое демо

<p align="center">
  <a href="https://drive.google.com/file/d/1frzw-HDZ3viQvLOQKsAJGc9bT0dXs68D/view">
    <img src="../../../screenshots/demo-cover.png" alt="Видео быстрого демо VL Intelligent" width="900"/>
  </a>
</p>

<p align="center">
  Нажмите на изображение обложки, чтобы посмотреть демо-видео.
</p>

---

## Возможности

| Функция | Описание |
| **Мульти-AI** | DeepSeek, Qwen, GPT, Claude, Gemini, Grok, Kimi, MiniMax — переключение в любой момент |

| **Студия стратегий** | Визуальный конструктор — источники монет, индикаторы, контроль рисков |
| **AI Арена дебатов** | Несколько AI обсуждают сделки (Бык vs Медведь vs Аналитик), голосуют, исполняют |
| **AI Соревнование** | AI соревнуются в реальном времени, рейтинг в таблице лидеров |
| **Telegram Агент** | Чат с торговым ассистентом — стриминг, вызов инструментов, память |
| **Лаборатория бэктеста** | Историческая симуляция с кривой капитала и метриками |
| **Панель управления** | Позиции в реальном времени, P/L, логи AI решений с Chain of Thought |

### Рынки

Криптовалюта · Акции США · Форекс · Металлы

### AI Модели (Режим API ключей)

| AI Модель | Статус | Получить API ключ |
| <img src="../../../web/public/icons/deepseek.svg" width="20" height="20" style="vertical-align: middle;"/> **DeepSeek** | ✅ | [Получить](https://platform.deepseek.com) |
| <img src="../../../web/public/icons/qwen.svg" width="20" height="20" style="vertical-align: middle;"/> **Qwen** | ✅ | [Получить](https://dashscope.console.aliyun.com) |
| <img src="../../../web/public/icons/openai.svg" width="20" height="20" style="vertical-align: middle;"/> **OpenAI (GPT)** | ✅ | [Получить](https://platform.openai.com) |
| <img src="../../../web/public/icons/claude.svg" width="20" height="20" style="vertical-align: middle;"/> **Claude** | ✅ | [Получить](https://console.anthropic.com) |
| <img src="../../../web/public/icons/gemini.svg" width="20" height="20" style="vertical-align: middle;"/> **Gemini** | ✅ | [Получить](https://aistudio.google.com) |
| <img src="../../../web/public/icons/grok.svg" width="20" height="20" style="vertical-align: middle;"/> **Grok** | ✅ | [Получить](https://console.x.ai) |
| <img src="../../../web/public/icons/kimi.svg" width="20" height="20" style="vertical-align: middle;"/> **Kimi** | ✅ | [Получить](https://platform.moonshot.cn) |
| <img src="../../../web/public/icons/minimax.svg" width="20" height="20" style="vertical-align: middle;"/> **MiniMax** | ✅ | [Получить](https://platform.minimaxi.com) |

---

## Установка

### Из исходников

```bash
# Требования: Go 1.21+, Node.js 18+, TA-Lib
# macOS: brew install ta-lib
# Ubuntu: sudo apt-get install libta-lib0-dev

go build -o vl-bin && ./vl-bin          # бэкенд
cd web && npm install && npm run dev  # фронтенд (новый терминал)
```

---

> **Предупреждение**: AI автоторговля несёт значительные риски. Рекомендуется только для обучения/исследований или тестирования малых сумм.

---

## Лицензия

[AGPL-3.0](../../../LICENSE)

