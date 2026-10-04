names rewritten to vl on 2026-09-30 (VL rename)
<h1 align="center">VL Intelligent</h1>

<p align="center">
  <strong>Ваш персональний AI торговий асистент.</strong><br/>

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
  <a href="../ru/README.md">Русский</a> ·
  <a href="README.md">Українська</a> ·
  <a href="../vi/README.md">Tiếng Việt</a>
</p>

---

VL Intelligent — це **автономний** AI торговий асистент з відкритим кодом. На відміну від традиційних AI інструментів, де потрібно вручну налаштовувати моделі, керувати API ключами та підключати джерела даних — AI у VL Intelligent **сам аналізує ринки, обирає моделі та отримує дані**. Нульове втручання людини. Ви задаєте стратегію, AI робить все інше.





Відкрийте **http://127.0.0.1:3000**. Готово.

---

## Швидке демо

<p align="center">
  <a href="https://drive.google.com/file/d/1frzw-HDZ3viQvLOQKsAJGc9bT0dXs68D/view">
    <img src="../../../screenshots/demo-cover.png" alt="Відео швидкого демо VL Intelligent" width="900"/>
  </a>
</p>

<p align="center">
  Натисніть на зображення обкладинки, щоб переглянути демо-відео.
</p>

---

## Можливості

| Функція | Опис |
| **Мульти-AI** | DeepSeek, Qwen, GPT, Claude, Gemini, Grok, Kimi, MiniMax — перемикання будь-коли |

| **Студія стратегій** | Візуальний конструктор — джерела монет, індикатори, контроль ризиків |
| **AI Арена дебатів** | Кілька AI обговорюють угоди, голосують, виконують |
| **AI Змагання** | AI змагаються в реальному часі, рейтинг у таблиці лідерів |
| **Telegram Агент** | Чат з торговим асистентом — стрімінг, виклик інструментів, пам'ять |
| **Лабораторія бектесту** | Історична симуляція з кривою капіталу та метриками |
| **Панель управління** | Позиції в реальному часі, P/L, логи AI рішень з Chain of Thought |

### Ринки

Криптовалюта · Акції США · Форекс · Метали

### AI Моделі (Режим API ключів)

| AI Модель | Статус | Отримати API ключ |
| <img src="../../../web/public/icons/deepseek.svg" width="20" height="20" style="vertical-align: middle;"/> **DeepSeek** | ✅ | [Отримати](https://platform.deepseek.com) |
| <img src="../../../web/public/icons/qwen.svg" width="20" height="20" style="vertical-align: middle;"/> **Qwen** | ✅ | [Отримати](https://dashscope.console.aliyun.com) |
| <img src="../../../web/public/icons/openai.svg" width="20" height="20" style="vertical-align: middle;"/> **OpenAI (GPT)** | ✅ | [Отримати](https://platform.openai.com) |
| <img src="../../../web/public/icons/claude.svg" width="20" height="20" style="vertical-align: middle;"/> **Claude** | ✅ | [Отримати](https://console.anthropic.com) |
| <img src="../../../web/public/icons/gemini.svg" width="20" height="20" style="vertical-align: middle;"/> **Gemini** | ✅ | [Отримати](https://aistudio.google.com) |
| <img src="../../../web/public/icons/grok.svg" width="20" height="20" style="vertical-align: middle;"/> **Grok** | ✅ | [Отримати](https://console.x.ai) |
| <img src="../../../web/public/icons/kimi.svg" width="20" height="20" style="vertical-align: middle;"/> **Kimi** | ✅ | [Отримати](https://platform.moonshot.cn) |
| <img src="../../../web/public/icons/minimax.svg" width="20" height="20" style="vertical-align: middle;"/> **MiniMax** | ✅ | [Отримати](https://platform.minimaxi.com) |

---

## Встановлення

### З вихідного коду

```bash
# Вимоги: Go 1.21+, Node.js 18+, TA-Lib
# macOS: brew install ta-lib
# Ubuntu: sudo apt-get install libta-lib0-dev

go build -o vl-bin && ./vl-bin          # бекенд
cd web && npm install && npm run dev  # фронтенд (новий термінал)
```

---

> **Попередження**: AI автоторгівля несе значні ризики. Рекомендується лише для навчання/досліджень або тестування малих сум.

---

## Ліцензія

[AGPL-3.0](../../../LICENSE)

