names rewritten to vl on 2026-09-30 (VL rename)
<h1 align="center">VL Intelligent</h1>

<p align="center">
  <strong>Trợ lý giao dịch AI cá nhân của bạn.</strong><br/>

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
  <a href="../uk/README.md">Українська</a> ·
  <a href="README.md">Tiếng Việt</a>
</p>

---

VL Intelligent là trợ lý giao dịch AI **tự chủ** mã nguồn mở. Không giống các công cụ AI truyền thống yêu cầu bạn cấu hình mô hình thủ công, quản lý API key và kết nối nguồn dữ liệu — AI của VL Intelligent **tự nhận diện thị trường, tự chọn mô hình và tự lấy dữ liệu**. Không cần con người can thiệp. Bạn chỉ cần đặt chiến lược, AI xử lý mọi thứ còn lại.





Mở **http://127.0.0.1:3000**. Xong.

---

## Demo nhanh

<p align="center">
  <a href="https://drive.google.com/file/d/1frzw-HDZ3viQvLOQKsAJGc9bT0dXs68D/view">
    <img src="../../../screenshots/demo-cover.png" alt="Video demo nhanh của VL Intelligent" width="900"/>
  </a>
</p>

<p align="center">
  Nhấp vào ảnh bìa để xem video demo.
</p>

---

## Tính năng

| Tính năng | Mô tả |
| **Đa AI** | DeepSeek, Qwen, GPT, Claude, Gemini, Grok, Kimi, MiniMax — chuyển đổi bất cứ lúc nào |

| **Strategy Studio** | Trình xây dựng trực quan — nguồn coin, chỉ báo, kiểm soát rủi ro |
| **AI Competition** | AI cạnh tranh thời gian thực, bảng xếp hạng hiệu suất |
| **Telegram Agent** | Chat với trợ lý giao dịch — streaming, gọi công cụ, bộ nhớ |
| **Dashboard** | Vị thế trực tiếp, P/L, nhật ký quyết định AI với Chain of Thought |

### Thị trường

Crypto · Cổ phiếu Mỹ · Forex · Kim loại

### Mô hình AI (Chế độ API Key)

| Mô hình AI | Trạng thái | Lấy API Key |
| <img src="../../../web/public/icons/deepseek.svg" width="20" height="20" style="vertical-align: middle;"/> **DeepSeek** | ✅ | [Lấy API Key](https://platform.deepseek.com) |
| <img src="../../../web/public/icons/qwen.svg" width="20" height="20" style="vertical-align: middle;"/> **Qwen** | ✅ | [Lấy API Key](https://dashscope.console.aliyun.com) |
| <img src="../../../web/public/icons/openai.svg" width="20" height="20" style="vertical-align: middle;"/> **OpenAI (GPT)** | ✅ | [Lấy API Key](https://platform.openai.com) |
| <img src="../../../web/public/icons/claude.svg" width="20" height="20" style="vertical-align: middle;"/> **Claude** | ✅ | [Lấy API Key](https://console.anthropic.com) |
| <img src="../../../web/public/icons/gemini.svg" width="20" height="20" style="vertical-align: middle;"/> **Gemini** | ✅ | [Lấy API Key](https://aistudio.google.com) |
| <img src="../../../web/public/icons/grok.svg" width="20" height="20" style="vertical-align: middle;"/> **Grok** | ✅ | [Lấy API Key](https://console.x.ai) |
| <img src="../../../web/public/icons/kimi.svg" width="20" height="20" style="vertical-align: middle;"/> **Kimi** | ✅ | [Lấy API Key](https://platform.moonshot.cn) |
| <img src="../../../web/public/icons/minimax.svg" width="20" height="20" style="vertical-align: middle;"/> **MiniMax** | ✅ | [Lấy API Key](https://platform.minimaxi.com) |

---

## Cài đặt

### Từ mã nguồn

```bash
# Yêu cầu: Go 1.21+, Node.js 18+, TA-Lib
# macOS: brew install ta-lib
# Ubuntu: sudo apt-get install libta-lib0-dev

go build -o vl-bin && ./vl-bin          # backend
cd web && npm install && npm run dev  # frontend (terminal mới)
```

---

> **Cảnh báo rủi ro**: Giao dịch tự động AI có rủi ro đáng kể. Chỉ nên sử dụng cho mục đích học tập/nghiên cứu hoặc số tiền nhỏ.

---

## License

[AGPL-3.0](../../../LICENSE)

