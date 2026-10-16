# Install — get from zero to a running SIM bot, in plain words

You are on a fresh **Windows 11 machine with WSL2** and **NinjaTrader 8** already
installed. You want the trading bot running in SIM (paper) mode, plus the
one-button updater. This is the whole thing.

## 0. One Windows setting first (2 minutes)

Open a PowerShell window (as you) and set WSL to **mirrored networking**, then
restart WSL:

```
wsl --shutdown
```
Add this to `C:\Users\<you>\.wslconfig`:

```
[wsl2]
networkingMode=mirrored
```

Then open your WSL Ubuntu shell again. This is what lets the bot (Linux) and
NinjaTrader (Windows) talk to each other over `127.0.0.1`. Without it, the bot
sees no bars and nothing trades.

## 1. Install Go, Node, git (once)

Inside WSL Ubuntu:

```
sudo apt update
sudo apt install -y git curl build-essential
```

- **Go** — install **1.26.9 or newer** from https://go.dev/dl/ (or your
  preferred method). Check with `go version`.
- **Node** — install **18 or newer** (22 is tested). nvm is fine. Check with
  `node --version`.

## 2. Get the code and run the installer

```
git clone <the repo URL> vl && cd vl
bash deploy/new-install.sh
```

The installer prints every step and refuses with a reason if anything is wrong
(not WSL, wrong Go/Node, no NinjaTrader folder). It:

- builds the bot and the web UI;
- writes your `.env` (it generates a secret key — never shown — and fills the
  required settings without touching anything you already set);
- prints **one sudo command** for auto-start — run it yourself;
- copies the NinjaTrader AddOn files into the Windows AddOns folder and tells
  you to press **F5** in NinjaTrader, then **restart NinjaTrader**.

`--dry-run` shows every step and changes nothing.

## 3. NinjaTrader (5 minutes, your hands only)

1. In NinjaTrader, press **F5** (compiles the AddOn).
2. Fully close NinjaTrader and open it again (AddOns do not hot-reload).

## 4. Start the bot and make your account

```
./vl-bin
```

- The bot creates its database on first start automatically.
- Open the UI at **http://localhost:3000**.
- The FIRST visit asks you to create the one and only account (type your email
  and a password). After that, registration is closed.
- Bind your **Sim101** account to a trader in Settings — SIM only. Live/funded
  accounts are blocked by the bot and cannot be enabled here.

## 5. The one-button updater

```
bash deploy/partner-first-time-setup.sh <your account email>
```

Run it once. After that, every release is one button in **Settings → Updates**
(plus F5 in NinjaTrader when the job says `nt8_updated`).

## 6. Is it working?

In the bot's terminal you should see a line ending in **`match=yes`** (the
AddOn's build id matches the bot) and **`hello handshake OK`**. The UI shows
bars and positions. `/api/health` answers on port 8080.

If you see `match=NO`, you copied the AddOn but skipped the F5 or the restart —
do step 3 again.

**SIM only.** This installs paper trading. There is no live-trading path here.
