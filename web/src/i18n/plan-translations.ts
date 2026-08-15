// P4 — Day-Plan / Plan Card i18n (self-contained feature module, mirrors the
// strategy-translations.ts pattern). Every entry carries all three languages
// (en / zh / id) — the TranslationEntry type makes a missing language a compile
// error. Controlled-vocab instruction verbs stay English in stored values;
// only the UI labels here translate (design-system Open Question #4).

import type { Language } from './translations'

interface TranslationEntry {
  en: string
  zh: string
  id: string
}

type PlanKey = keyof typeof planStrings

export const planStrings = {
  // ── header ──
  title: { en: "Today's Plan", zh: '今日计划', id: 'Rencana Hari Ini' },
  edit: { en: 'Edit', zh: '编辑', id: 'Ubah' },
  reread: { en: 'Re-read', zh: '重新解读', id: 'Baca ulang' },
  approve: { en: 'Approve', zh: '批准', id: 'Setujui' },
  askPlanner: { en: 'Ask planner', zh: '询问规划师', id: 'Tanya perencana' },

  // ── lifecycle chips ──
  lifecycleActive: { en: 'ACTIVE', zh: '生效中', id: 'AKTIF' },
  lifecycleExpired: { en: 'EXPIRED', zh: '已过期', id: 'KEDALUWARSA' },
  lifecycleDied: { en: 'DIED', zh: '已失效', id: 'MATI' },
  lifecycleSuperseded: { en: 'SUPERSEDED', zh: '已替换', id: 'DIGANTI' },

  // ── card states ──
  noPlanYet: { en: 'No plan yet', zh: '尚无计划', id: 'Belum ada rencana' },
  noPlanYetHint: {
    en: 'The planner arms at the session open.',
    zh: '规划师将在时段开盘时生成计划。',
    id: 'Perencana aktif saat sesi dibuka.',
  },
  night: {
    en: 'Night — markets quiet',
    zh: '夜间 — 市场清淡',
    id: 'Malam — pasar sepi',
  },
  nightHint: {
    en: 'No active session. The plan resumes at the next session open.',
    zh: '当前无活跃时段，计划将于下一时段开盘恢复。',
    id: 'Tidak ada sesi aktif. Rencana lanjut pada sesi berikutnya.',
  },
  disabled: {
    en: 'Day Plan off',
    zh: '日计划已关闭',
    id: 'Rencana Harian nonaktif',
  },
  disabledHint: {
    en: 'Enable Day Plan in Strategy Studio.',
    zh: '在策略工作室中启用日计划。',
    id: 'Aktifkan Rencana Harian di Strategy Studio.',
  },
  errorFailClosed: {
    en: 'Plan read failed — sitting out',
    zh: '计划解读失败 — 本时段观望',
    id: 'Gagal baca rencana — libur sesi ini',
  },
  errorHint: {
    en: 'Fail-closed: no valid plan this session.',
    zh: '安全失败：本时段无有效计划。',
    id: 'Fail-closed: tidak ada rencana valid sesi ini.',
  },
  reading: {
    en: 'Reading the tape…',
    zh: '正在解读盘面…',
    id: 'Membaca pita…',
  },
  loading: { en: 'Loading plan…', zh: '加载计划中…', id: 'Memuat rencana…' },

  // ── bias ──
  bias: { en: 'Bias', zh: '倾向', id: 'Bias' },
  biasLong: { en: 'LONG', zh: '做多', id: 'LONG' },
  biasShort: { en: 'SHORT', zh: '做空', id: 'SHORT' },
  biasNeutral: { en: 'NEUTRAL', zh: '中性', id: 'NETRAL' },
  conviction: { en: 'conviction', zh: '信心', id: 'keyakinan' },
  flipPrefix: { en: 'Flips', zh: '反转条件', id: 'Berbalik' },

  // ── badges / banners ──
  warming: { en: 'WARMING {n}', zh: '预热中 {n}', id: 'PEMANASAN {n}' },
  uncalibrated: { en: 'UNCALIBRATED', zh: '未校准', id: 'BELUM DIKALIBRASI' },
  advisoryMode: {
    en: 'Advisory mode — the plan informs, the AI still decides.',
    zh: '顾问模式 — 计划仅供参考，仍由 AI 决策。',
    id: 'Mode saran — rencana memberi info, AI tetap memutuskan.',
  },
  rereadsLeft: {
    en: '{n} re-reads left',
    zh: '剩余 {n} 次重读',
    id: '{n} baca ulang tersisa',
  },

  // ── levels ──
  keyLevels: { en: 'Key Levels', zh: '关键价位', id: 'Level Kunci' },
  colPrice: { en: 'Price', zh: '价格', id: 'Harga' },
  colProvenance: { en: 'Source', zh: '来源', id: 'Sumber' },
  colGrade: { en: 'Grade', zh: '评级', id: 'Nilai' },
  colFresh: { en: 'State', zh: '状态', id: 'Status' },
  colInstruction: { en: 'Instruction', zh: '指令', id: 'Instruksi' },
  colDistance: { en: 'Dist', zh: '距离', id: 'Jarak' },
  freshFresh: { en: 'fresh', zh: '新鲜', id: 'segar' },
  freshTested: { en: 'tested', zh: '已测试', id: 'teruji' },
  freshConsumed: { en: 'consumed', zh: '已消耗', id: 'terpakai' },
  ownerLevel: { en: 'Owner level', zh: '所有者价位', id: 'Level pemilik' },
  hasNote: {
    en: 'Has a note for the AI',
    zh: '含给 AI 的备注',
    id: 'Ada catatan untuk AI',
  },
  noLevels: {
    en: 'No levels in this plan',
    zh: '此计划无价位',
    id: 'Tidak ada level di rencana ini',
  },

  // ── scenarios ──
  scenarios: { en: 'Scenarios', zh: '情景', id: 'Skenario' },
  statusArmed: { en: 'armed', zh: '待触发', id: 'siap' },
  statusWaiting: { en: 'waiting', zh: '等待中', id: 'menunggu' },
  statusTriggered: { en: 'triggered', zh: '已触发', id: 'terpicu' },
  statusInvalidated: { en: 'invalidated', zh: '已失效', id: 'batal' },
  statusExpired: { en: 'expired', zh: '已过期', id: 'kedaluwarsa' },
  targets: { en: 'targets', zh: '目标', id: 'target' },
  invalidates: { en: 'invalid', zh: '失效', id: 'batal' },

  // ── rules ──
  noTrade: { en: 'No-trade', zh: '禁止交易', id: 'Larangan trade' },
  planDies: { en: 'Plan dies if', zh: '计划失效条件', id: 'Rencana mati jika' },

  // ── footer ──
  dayType: { en: 'Day type', zh: '日型', id: 'Tipe hari' },
  model: { en: 'Model', zh: '模型', id: 'Model' },

  // ── sessions / timeline ──
  sessionAsia: { en: 'Asia', zh: '亚洲', id: 'Asia' },
  sessionLondon: { en: 'London', zh: '伦敦', id: 'London' },
  sessionNY: { en: 'New York', zh: '纽约', id: 'New York' },
  now: { en: 'now', zh: '当前', id: 'sekarang' },
  killzone: { en: 'killzone', zh: '狙击时段', id: 'killzone' },
  dstWarning: {
    en: 'DST shift — London window may be off by 1h',
    zh: '夏令时切换 — 伦敦时段可能偏差 1 小时',
    id: 'Pergeseran DST — jendela London bisa meleset 1 jam',
  },
  tabReading: { en: 'reading', zh: '解读中', id: 'membaca' },
  tabNight: { en: 'night', zh: '夜间', id: 'malam' },

  // ── handover banner ──
  handoverExpired: {
    en: '{session} plan expired',
    zh: '{session} 计划已过期',
    id: 'Rencana {session} kedaluwarsa',
  },
  handoverReading: {
    en: 'Reading {session}…',
    zh: '正在解读 {session}…',
    id: 'Membaca {session}…',
  },
  handoverBorn: {
    en: '{session} plan is live',
    zh: '{session} 计划已生效',
    id: 'Rencana {session} aktif',
  },
  handoverReadFailed: {
    en: '{session} read failed — sitting out',
    zh: '{session} 解读失败 — 观望',
    id: 'Baca {session} gagal — libur',
  },

  // ── alert center ──
  alerts: { en: 'Alerts', zh: '提醒', id: 'Peringatan' },
  noAlerts: { en: 'No alerts', zh: '暂无提醒', id: 'Tidak ada peringatan' },
  markRead: { en: 'Mark read', zh: '标记已读', id: 'Tandai dibaca' },
  markAllRead: {
    en: 'Mark all read',
    zh: '全部标记已读',
    id: 'Tandai semua dibaca',
  },
  unread: { en: '{n} unread', zh: '{n} 条未读', id: '{n} belum dibaca' },
  bellLabel: {
    en: 'Alerts, {n} unread',
    zh: '提醒，{n} 条未读',
    id: 'Peringatan, {n} belum dibaca',
  },
  dismiss: { en: 'Dismiss', zh: '关闭', id: 'Tutup' },

  // ── Studio Day Plan block ──
  dayPlanBlock: { en: 'Day Plan', zh: '日计划', id: 'Rencana Harian' },
  enableDayPlan: {
    en: 'Enable Day Plan',
    zh: '启用日计划',
    id: 'Aktifkan Rencana Harian',
  },
  plannerModel: { en: 'Planner model', zh: '规划模型', id: 'Model perencana' },
  planMode: { en: 'Plan mode', zh: '计划模式', id: 'Mode rencana' },
  modeAdvisory: { en: 'ADVISORY', zh: '顾问', id: 'SARAN' },
  modeDirection: { en: 'DIRECTION', zh: '定向', id: 'ARAH' },
  modeStrict: { en: 'STRICT', zh: '严格', id: 'KETAT' },
  plannerReads: {
    en: 'Planner reads',
    zh: '规划师读取',
    id: 'Perencana membaca',
  },
  autoLabel: { en: 'AUTO', zh: '自动', id: 'OTOMATIS' },
  autoTooltip: {
    en: 'auto-computed — not a setting',
    zh: '自动计算 — 非可配置项',
    id: 'dihitung otomatis — bukan pengaturan',
  },
  regime: { en: 'Regime', zh: '市况', id: 'Rezim' },
  filters: { en: 'Filters', zh: '筛选', id: 'Filter' },
  proximity: { en: 'Proximity', zh: '邻近度', id: 'Kedekatan' },
  maxLevels: { en: 'Max levels', zh: '最大价位数', id: 'Maks level' },
  maxScenarios: { en: 'Max scenarios', zh: '最大情景数', id: 'Maks skenario' },
  maxReplans: {
    en: 'Max re-plans',
    zh: '最大重规划数',
    id: 'Maks rencana ulang',
  },
  minGrade: { en: 'Min grade', zh: '最低评级', id: 'Nilai min' },
  maxTrades: { en: 'Max trades', zh: '最大交易数', id: 'Maks trade' },
  acceptance: { en: 'Acceptance', zh: '接受确认', id: 'Penerimaan' },
  approval: {
    en: 'Approval required',
    zh: '需要批准',
    id: 'Perlu persetujuan',
  },
  digest: { en: 'Digest', zh: '摘要', id: 'Ringkasan' },
  sessionsHeader: { en: 'Sessions', zh: '交易时段', id: 'Sesi' },
  inherit: { en: 'inherit', zh: '继承', id: 'warisi' },
  override: { en: 'override', zh: '覆盖', id: 'ganti' },
  windows: { en: 'Windows', zh: '时间窗口', id: 'Jendela' },

  // ── P5.2 edit sheet + bulk add (LABELS translate; stored tokens stay English) ──
  editLevel: { en: 'Edit level', zh: '编辑价位', id: 'Ubah level' },
  addLevel: { en: 'Add level', zh: '添加价位', id: 'Tambah level' },
  priceRange: {
    en: 'Price / zone range',
    zh: '价格 / 区间',
    id: 'Harga / rentang zona',
  },
  levelType: { en: 'Type', zh: '类型', id: 'Tipe' },
  instructionLabel: {
    en: 'Instruction (what the AI does here)',
    zh: '指令（AI 在此如何操作）',
    id: 'Instruksi (aksi AI di sini)',
  },
  gradeLabel: { en: 'Grade', zh: '评级', id: 'Nilai' },
  noteLabel: {
    en: 'Note (goes to the AI)',
    zh: '备注（发送给 AI）',
    id: 'Catatan (untuk AI)',
  },
  scenarioTag: { en: 'Scenario tag', zh: '情景标签', id: 'Tag skenario' },
  newPlay: { en: '＋ new play', zh: '＋ 新剧本', id: '＋ play baru' },
  save: { en: 'Save', zh: '保存', id: 'Simpan' },
  cancel: { en: 'Cancel', zh: '取消', id: 'Batal' },
  deleteLevel: { en: 'Delete', zh: '删除', id: 'Hapus' },
  armorWarn: {
    en: '⛔ armor check runs on save — a price ~8×ATR away gets rejected, same as the AI’s would',
    zh: '⛔ 保存时进行护栏检查 — 距价约 8×ATR 的价位会被拒绝，与 AI 的一样',
    id: '⛔ cek armor saat simpan — harga ~8×ATR jauh ditolak, sama seperti AI',
  },
  bulkAdd: { en: 'Bulk add', zh: '批量添加', id: 'Tambah massal' },
  bulkAddHint: {
    en: 'One level per line: price [type] [note]',
    zh: '每行一个价位：价格 [类型] [备注]',
    id: 'Satu level per baris: harga [tipe] [catatan]',
  },
  preview: { en: 'Preview', zh: '预览', id: 'Pratinjau' },
  overlayApplied: {
    en: 'Plan updated',
    zh: '计划已更新',
    id: 'Rencana diperbarui',
  },
  saveFailed: { en: 'Save rejected', zh: '保存被拒', id: 'Simpan ditolak' },

  // ── level type labels (values stay English tokens) ──
  typeDzone: { en: 'D-zone', zh: '需求区', id: 'Zona-D' },
  typeSzone: { en: 'S-zone', zh: '供给区', id: 'Zona-S' },
  typeLevel: { en: 'Level', zh: '价位', id: 'Level' },
  typeLiquidity: { en: 'Liquidity', zh: '流动性', id: 'Likuiditas' },
  typeMyLevel: { en: 'My level', zh: '我的价位', id: 'Level saya' },

  // ── instruction verb labels ──
  instrSweepReclaim: {
    en: 'sweep+reclaim = entry',
    zh: '扫单+夺回 = 进场',
    id: 'sweep+reclaim = masuk',
  },
  instrWatchReclaim: {
    en: 'watch reclaim',
    zh: '观察夺回',
    id: 'pantau reclaim',
  },
  instrFadeFirst: {
    en: 'fade first touch',
    zh: '首次触及反向',
    id: 'fade sentuhan pertama',
  },
  instrTargetOnly: { en: 'target only', zh: '仅作目标', id: 'target saja' },
  instrMagnet: {
    en: 'magnet — no entries',
    zh: '磁吸 — 不进场',
    id: 'magnet — tanpa entri',
  },
  instrNoTouch: { en: 'no touch', zh: '勿触及', id: 'jangan sentuh' },

  // ── P5.3 conflict chip ──
  conflict: { en: 'conflict', zh: '冲突', id: 'konflik' },
  conflictHint: {
    en: 'Owner wins on execution; the AI element is ghosted (both logged).',
    zh: '执行以所有者为准；AI 元素被虚化（均记录）。',
    id: 'Pemilik menang saat eksekusi; elemen AI diredupkan (keduanya dicatat).',
  },

  // ── P5.4 Ask-Planner ──
  askPlannerTitle: {
    en: 'Ask Planner',
    zh: '询问规划师',
    id: 'Tanya Perencana',
  },
  askEvidence: { en: 'Evidence', zh: '依据', id: 'Bukti' },
  askYourPoint: { en: 'Your point', zh: '你的观点', id: 'Poin Anda' },
  askVerdict: { en: 'Verdict', zh: '裁决', id: 'Putusan' },
  clsNewInfo: { en: 'NEW INFO', zh: '新信息', id: 'INFO BARU' },
  clsBare: { en: 'BARE DISAGREEMENT', zh: '无据反对', id: 'BEDA TANPA BUKTI' },
  vDefend: { en: 'DEFEND', zh: '坚持', id: 'PERTAHANKAN' },
  vConcede: { en: 'CONCEDE', zh: '让步', id: 'MENGALAH' },
  vMerge: { en: 'PROPOSE-MERGE', zh: '提议合并', id: 'USUL-GABUNG' },
  applyMerge: { en: 'Apply merge', zh: '应用合并', id: 'Terapkan gabungan' },
  keepAsIs: { en: 'Keep as-is', zh: '保持不变', id: 'Biarkan' },
  patchNote: {
    en: 'nothing changes until you Apply',
    zh: '在你应用前不会改变任何内容',
    id: 'tidak ada perubahan sampai Anda Terapkan',
  },
  askApplied: {
    en: 'Applied — card updated',
    zh: '已应用 — 计划已更新',
    id: 'Diterapkan — kartu diperbarui',
  },
  askPlaceholder: {
    en: "ask about today's plan… (any language)",
    zh: '询问今日计划…（任何语言）',
    id: 'tanya rencana hari ini… (bahasa apa saja)',
  },
  send: { en: 'Send', zh: '发送', id: 'Kirim' },
  qcWhyLevel: {
    en: 'why this level?',
    zh: '为何是这个价位？',
    id: 'kenapa level ini?',
  },
  qcWhatKills: {
    en: 'what kills the plan?',
    zh: '什么会让计划失效？',
    id: 'apa yang mematikan rencana?',
  },
  sycophancyKPI: {
    en: 'defends-on-bare',
    zh: '无据坚持',
    id: 'pertahankan-tanpa-bukti',
  },
  thinking: { en: 'thinking…', zh: '思考中…', id: 'berpikir…' },

  // ── P5.5 / P5.6 badges ──
  adherence: { en: 'Adherence', zh: '计划遵守', id: 'Kepatuhan' },
  gpa: { en: 'GPA', zh: '平均分', id: 'IPK' },
  tradeReview: { en: 'Trade review', zh: '交易复盘', id: 'Tinjauan trade' },
  beatsRandom: { en: 'beats random', zh: '优于随机', id: 'kalahkan acak' },
  noEdge: { en: 'no edge', zh: '无优势', id: 'tanpa edge' },
  statsWarming: {
    en: 'WARMING {n}/{target}',
    zh: '预热中 {n}/{target}',
    id: 'PEMANASAN {n}/{target}',
  },
} satisfies Record<string, TranslationEntry>

// tp(key, lang, params?) — resolve a plan string, fall back to English, then to
// the key itself; supports {param} interpolation (e.g. warming "{n}" → "3/10").
export function tp(
  key: PlanKey,
  lang: Language,
  params?: Record<string, string | number>
): string {
  const entry = planStrings[key]
  let out = (entry && (entry[lang] ?? entry.en)) || key
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      out = out.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v))
    }
  }
  return out
}

export type { PlanKey }
