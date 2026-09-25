import { GUIDE_BUILT_REV, type GuideSection } from '../types'

export const updates: GuideSection = {
  id: 'updates',
  num: 16,
  title: 'Updates',
  tagline:
    'The one-button update, its hold, its gate, and what a receipt proves.',
  asBuiltRev: GUIDE_BUILT_REV,
  blocks: [
    { kind: 'h', text: 'The Updates page (Settings → Updates)' },
    {
      kind: 'p',
      text: 'Settings → Updates is the one screen for updating the bot. It shows what is running now (the revision, the guide build, the AddOn build the maintenance acknowledgement names), the update button, the maintenance hold and the installation gate, the update job, and the native-history readiness. It refreshes itself every ten seconds. Everything on it is READ from the bot’s own API and nothing else: a value the bot cannot tell you prints n/a, and an update job that has not run is absent — the page says "no update job", it does not draw an empty timeline.',
    },
    { kind: 'h', text: 'What each update-button state means' },
    {
      kind: 'p',
      text: 'Update now: the button is idle and the bot has not reported an available update or a refusal. Checking…: the page is asking the bot whether an update exists. Installing…: an install was accepted and is running. Retry: the last attempt was refused and the page shows the bot’s exact reason. Up to date: the bot answered that no update is available. Blocked: the bot says installs are not enabled on this build. Until the adversarial review of the update authorization closes, the install button itself is disabled and reads "install authorization under review" — no install can be started from the page while that is in force. The header badge polls the same endpoint every 60 seconds and reads Unknown whenever the bot has not affirmed a state.',
    },
    { kind: 'h', text: 'The hold and the gate' },
    {
      kind: 'p',
      text: 'The maintenance panel shows the hold state (clear, held, unreadable, or unconfigured), whether the bot has drained its in-flight sends, and the AddOn’s acknowledgement of the current hold. Until an acknowledgement arrives the panel says "no ack yet" — it never turns a missing ack into "false". The installation gate is one verdict for the whole installation, not one trader: every leg the update needs must pass, and each leg’s exact reason text is shown. A leg that could not be evaluated fails. The hold is written and cleared only by the local operator tool or the updater itself — never by this page.',
    },
    { kind: 'h', text: 'What a receipt proves — and what it cannot' },
    {
      kind: 'p',
      text: 'A receipt is the recorded proof of a completed update job: the states the job passed through, its timestamps, and whether it ended complete, rolled back, or in need of recovery. A receipt from an AddOn that was never restarted, or from an AddOn whose build does not match the release, never satisfies verification — a pre-restart receipt proves the install reached a machine, not that the machine is running the new code. The page shows the receipt download link only when the API offers one.',
    },
    { kind: 'h', text: 'Native history readiness' },
    {
      kind: 'p',
      text: 'The readiness panel reports how far the native 4-hour history load has got. PivotWindow is read from the resolved strategy settings; the completed-bar target is PivotWindow + 4. Where the bot exposes no number — the completed count, the last progress, or a paused-while-loading flag — the panel prints n/a rather than computing or guessing one in the browser. Reload history asks the bot for a deep bars backfill through the existing backfill route.',
    },
    { kind: 'h', text: 'Enrolling this box for updates' },
    {
      kind: 'p',
      text: 'Updates stay OFF until the installation’s update administrator is enrolled, on the box itself, as the bot’s own user: `go run ./cmd/updater-bootstrap --install-dir <the bot’s folder> enroll <your exact account email>`, then type the confirmation it asks for. No page and no API can enroll or reset it; the Updates routes only READ the two enrollment files on each request. Once enrolled, the Updates routes answer only a browser on this box that opens the bot directly at 127.0.0.1 or localhost — never through a proxy, a tunnel or the LAN address — signed in as the enrolled account. Every refusal is the same "forbidden"; the bot logs the reason (🔒 [updates] refused …). In this build nothing installs yet: the release verifier refuses every release and the install button stays disabled. The full procedure, including un-enrolling, is docs/superpowers/runbooks/2026-09-24-m3-update-enrollment.md.',
    },
    { kind: 'h', text: 'Changing the password (Settings → Account)' },
    {
      kind: 'p',
      text: 'Changing the password needs your CURRENT password as well as the new one: the bot checks it against the stored password and refuses a wrong one ("current password is incorrect", shown under the form). A signed-in session alone can no longer set a new password. Machine tokens — the Telegram bot’s and the gate-jwt tool’s — can never change a password, reset the account, log out, change the Telegram settings or reach Updates, whatever they carry (a machine token has no session to end, and a refused logout leaves it working). A wrong current password is answered after a one-second pause and counted; it shows in the refusals panel as “Wrong current password (password change)”. A password change also un-enrolls Updates: every Updates route answers 403 until you re-enroll on the box with `go run ./cmd/updater-bootstrap enroll --replace <email>` (confirm you can sign in with the new password first). Locked out? Password reset by email is disabled; the fix is ONE statement on the box that sets the new password hash AND updated_at together (the runbook has it). Moving updated_at is what signs out every session issued before the reset — a hash-only change would leave them valid.',
    },
    {
      kind: 'p',
      text: 'A password change ends EVERY session signed in before it — on every page, not only on Settings — including the one you changed it from: the page signs you out and you sign in again with the new password. A token issued in the same second as the change is ended too: a sign-in within that second succeeds, but its new session is refused on first use and the page signs you out — sign in again. A session token whose account no longer exists (after a reset-account) is refused everywhere. The Telegram bot acts for the FIRST account registered on the box: when that account’s password changes, the bot notices its own token was ended and mints a new one before its next reply, and that chat’s conversation memory starts fresh (a change on any other account leaves the bot’s token alone). A session token stamped more than a minute ahead of the box’s clock is refused everywhere, so if the box’s clock steps BACK by more than a minute, sessions signed in during the skipped time are refused (the page signs you out) until the clock catches up — a step of a minute or less costs nothing, and the same minute of slack lets a token act for up to a minute past its 24-hour expiry. Thrown out right after a password change, with the bot’s log saying “credential epoch is Ns in the future — clock stepped back; sign-in refused until then”? The box’s clock moved backwards after the change: sign-in works again once the clock passes that moment (fix the clock, or wait N seconds).',
    },
  ],
}
