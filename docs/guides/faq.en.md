names rewritten to vl on 2026-09-30 (VL rename)
# Frequently Asked Questions (FAQ)

Quick answers to common questions. For detailed troubleshooting, see [Troubleshooting Guide](TROUBLESHOOTING.md).

---

## General Questions

### What is VL?
VL is an AI-powered trading bot that uses large language models (LLMs) to make trading decisions on futures markets.

### Which exchanges are supported?

### Is VL profitable?
AI trading is **experimental** and **not guaranteed** to be profitable. Always start with small amounts and never invest more than you can afford to lose.

### Can I run multiple traders simultaneously?
Yes! VL supports running multiple traders with different configurations, AI models, and trading strategies.

---

## Setup & Configuration

### What are the system requirements?
- **OS**: Linux, macOS, or Windows (Docker recommended)
- **RAM**: 2GB minimum, 4GB recommended
- **Disk**: 1GB for application + logs
- **Network**: Stable internet connection

### Do I need coding experience?
No! VL has a web UI for all configuration. However, basic command line knowledge helps with setup and troubleshooting.

---

## Trading Questions

### Why isn't my trader making any trades?
Common reasons:
- AI decided to "wait" due to market conditions
- Insufficient balance or margin
- Position limits reached (default: max 3 positions)
- See detailed diagnostics in [Troubleshooting Guide](TROUBLESHOOTING.md#-ai-always-says-wait--hold)

### How often does the AI make decisions?
Configurable! Default is every **3-5 minutes**. Too frequent = overtrading, too slow = missed opportunities.

### Can I customize the trading strategy?
Yes! You can:
- Adjust leverage settings
- Modify coin selection pool
- Change decision intervals
- Customize system prompts (advanced)

### What's the maximum number of concurrent positions?
Default: **3 positions**. This is a soft limit defined in the AI prompt, not hard-coded. See `decision/engine.go:266`.

---

## Technical Issues

### Backend won't start / Port already in use

**Solution**:
```bash
# Check what's using port 8080
lsof -i :8080

# Change port in .env
VL_BACKEND_PORT=8081
```

---

### Frontend shows "Loading..." forever

**Quick Check**:
```bash
# Is backend running?
curl http://localhost:8080/api/health

# Should return: {"status":"ok"}
```

If not, check [Troubleshooting Guide](TROUBLESHOOTING.md#-frontend-cant-connect-to-backend).

---

### Database locked error

**Solution**:
```bash
# Stop all VL processes
docker compose down
# OR
pkill vl

# Restart
docker compose up -d
```

---

## AI & Model Questions

### Which AI models are supported?
- **DeepSeek** (recommended for cost/performance)
- **Qwen** (Alibaba Cloud Tongyi Qianwen)
- **Custom OpenAI-compatible APIs** (can be used for OpenAI, Claude via proxy, or other providers)

### How much do API calls cost?
Depends on your model and decision frequency:
- **DeepSeek**: ~$0.10-0.50 per day (1 trader, 5min intervals)
- **Qwen**: ~$0.20-0.80 per day
- **Custom API** (e.g., OpenAI GPT-4): ~$2-5 per day

*Estimates based on typical usage. Actual costs vary by provider and usage.*

### Can I use multiple AI models?
Yes! Each trader can use a different AI model. You can even A/B test different models.

### Does the AI learn from its mistakes?
Yes, to some extent. VL provides historical performance feedback in each decision prompt, allowing the AI to adjust its strategy.

---

## Data & Privacy

### Where is my data stored?
All data is stored **locally** in PostgreSQL (Docker volume `postgres_data`) plus:
- `decision_logs/` - AI decision records

### Is my API key secure?
API keys are stored in local databases. Never share your databases or `.env` files. We recommend using API keys with IP whitelist restrictions.

### Can I export my trading history?
Yes! Use `pg_dump` or `psql` to export data:
```bash
docker compose exec postgres \
  psql -U vl -d vl -c "SELECT * FROM trades;"
```

---

## Troubleshooting

### Where can I find detailed troubleshooting?
See the comprehensive [Troubleshooting Guide](TROUBLESHOOTING.md) for:
- Step-by-step diagnostics
- Log collection methods
- Common error solutions
- Emergency reset procedures

### How do I report a bug?
1. Check [Troubleshooting Guide](TROUBLESHOOTING.md) first
2. Search [existing issues](upstream github link (removed in the VL rename))
3. If not found, use our [Bug Report Template](../../.github/ISSUE_TEMPLATE/bug_report.md)

### Where can I get help?

---

## Contributing

### Can I contribute to VL?
Yes! We welcome contributions:
- Bug fixes and features
- Documentation improvements
- Translations
- See [Contributing Guide](../CONTRIBUTING.md)

### How do I suggest new features?
Open a [Feature Request](upstream github link (removed in the VL rename)) with your idea!

---

**Last Updated:** 2025-11-02
