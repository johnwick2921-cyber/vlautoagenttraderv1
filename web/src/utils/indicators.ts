// 技术指标计算工具

export interface Kline {
  time: number
  open: number
  high: number
  low: number
  close: number
  volume?: number
}

// 简单移动平均线 (SMA)
export function calculateSMA(
  data: Kline[],
  period: number
): Array<{ time: number; value: number }> {
  const result: Array<{ time: number; value: number }> = []

  for (let i = period - 1; i < data.length; i++) {
    let sum = 0
    for (let j = 0; j < period; j++) {
      sum += data[i - j].close
    }
    result.push({
      time: data[i].time,
      value: sum / period,
    })
  }

  return result
}

// 指数移动平均线 (EMA)
export function calculateEMA(
  data: Kline[],
  period: number
): Array<{ time: number; value: number }> {
  const result: Array<{ time: number; value: number }> = []
  const multiplier = 2 / (period + 1)

  // 第一个EMA值使用SMA
  let ema = 0
  for (let i = 0; i < period; i++) {
    ema += data[i].close
  }
  ema = ema / period
  result.push({ time: data[period - 1].time, value: ema })

  // 后续EMA值
  for (let i = period; i < data.length; i++) {
    ema = (data[i].close - ema) * multiplier + ema
    result.push({ time: data[i].time, value: ema })
  }

  return result
}

// 布林带
export interface BollingerBands {
  time: number
  upper: number
  middle: number
  lower: number
}

export function calculateBollingerBands(
  data: Kline[],
  period = 20,
  stdDev = 2
): BollingerBands[] {
  const result: BollingerBands[] = []

  for (let i = period - 1; i < data.length; i++) {
    // 计算SMA
    let sum = 0
    for (let j = 0; j < period; j++) {
      sum += data[i - j].close
    }
    const sma = sum / period

    // 计算标准差
    let variance = 0
    for (let j = 0; j < period; j++) {
      variance += Math.pow(data[i - j].close - sma, 2)
    }
    const std = Math.sqrt(variance / period)

    result.push({
      time: data[i].time,
      upper: sma + stdDev * std,
      middle: sma,
      lower: sma - stdDev * std,
    })
  }

  return result
}
