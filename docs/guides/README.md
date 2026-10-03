names rewritten to vl on 2026-09-30 (VL rename)
# 📘 VL User Guides

**Language:** [English](README.md) | [中文](README.zh-CN.md)

Comprehensive guides to help you use VL effectively.

---

## 📚 Available Guides

### 🔧 Basic Usage

| Guide | Description | Status |
|-------|-------------|--------|
| [FAQ (English)](faq.en.md) | Frequently asked questions | ✅ Available |
| [FAQ (中文)](faq.zh-CN.md) | 常见问题解答 | ✅ Available |
| Configuration Guide | Advanced settings and options | 🚧 Coming Soon |
| Trading Strategies | AI trading strategy examples | 🚧 Coming Soon |

---

## 🐛 Troubleshooting

### Common Issues

**Issue: TA-Lib not found**
```bash
# macOS
brew install ta-lib

# Ubuntu/Debian
sudo apt-get install libta-lib0-dev
```

**Issue: Precision error**
- System auto-handles LOT_SIZE from exchange
- Check network connection
- Verify exchange API is accessible

**Issue: AI API timeout**
- Check API key validity
- Verify network connection
- Check API balance/credits
- Timeout is set to 120 seconds

**Issue: Frontend can't connect**
- Ensure backend is running (http://localhost:8080)
- Check if port 8080 is available
- Check browser console for errors

---

## 📖 Usage Tips

### Best Practices

**1. Risk Management**
- Start with small size (1 MNQ contract)
- Use subaccounts for additional safety
- Set reasonable leverage limits
- Monitor daily loss limits

**2. Performance Monitoring**
- Check decision logs regularly
- Analyze win rate and profit factor
- Review AI reasoning (Chain of Thought)
- Track equity curve trends

**3. Configuration**
- Test on testnet first
- Gradually increase trading amounts
- Adjust scan intervals (3-5 minutes recommended)
- Use default coin list for beginners

---

## 🎯 Advanced Topics

### Multi-Trader Competition
Run multiple AI models simultaneously:
- Qwen vs DeepSeek head-to-head
- Compare performance in real-time
- Identify best-performing strategies

### Symbol Selection
- Static symbol list in Strategy Studio
- Filter by liquidity and volume

### Venue
- NinjaTrader 8 (CME futures — SIM execution)

---

## 📊 Understanding Metrics

### Key Performance Indicators

**Win Rate**
- Percentage of profitable trades
- Target: >50% for consistent profit

**Profit Factor**
- Ratio of gross profit to gross loss
- Target: >1.5 (1.5:1 or better)

**Sharpe Ratio**
- Risk-adjusted return measure
- Higher is better (>1.0 is good)

**Maximum Drawdown**
- Largest peak-to-trough decline
- Keep under 20% for safety

---

## 🔗 Related Documentation

- [Getting Started (EN)](../getting-started/README.md) - Initial setup
- [Getting Started (中文)](../getting-started/README.zh-CN.md) - 初始设置
- [Community](../community/README.md) - Contributing and bounties
- [FAQ (English)](faq.en.md) - Common questions
- [FAQ (中文)](faq.zh-CN.md) - 常见问题

---

## 🆘 Need Help?

**Can't find what you need?**
- 💬 [Telegram Community](upstream telegram link (removed in the VL rename))
- 🐛 [GitHub Issues](upstream github link (removed in the VL rename))
- 🐦 [Twitter upstream x link (removed in the VL rename)](upstream x link (removed in the VL rename))

---

[← Back to Documentation Home](../README.md)
