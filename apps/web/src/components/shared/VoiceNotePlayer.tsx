import { useCallback, useEffect, useRef, useState } from 'react'
import { Pause, Play } from 'lucide-react'
import { cn } from '@/lib/utils'

const BAR_COUNT = 54

function fakeWaveBars() {
  return Array.from({ length: BAR_COUNT }, (_, i) => 0.22 + Math.abs(Math.sin(i * 0.45)) * 0.55)
}

/** Formato estilo WhatsApp: m:ss ou h:mm:ss */
function fmtTime(sec: number) {
  if (!Number.isFinite(sec) || sec < 0) return '0:00'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = Math.floor(sec % 60)
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }
  return `${m}:${s.toString().padStart(2, '0')}`
}

type Props = {
  src: string
  outbound: boolean
}

export function VoiceNotePlayer({ src, outbound }: Props) {
  const audioRef = useRef<HTMLAudioElement>(null)
  const waveRef = useRef<HTMLDivElement>(null)
  const [playing, setPlaying] = useState(false)
  const [current, setCurrent] = useState(0)
  const [duration, setDuration] = useState(0)
  const [bars, setBars] = useState<number[] | null>(null)
  const [decodeErr, setDecodeErr] = useState(false)
  const [elementErr, setElementErr] = useState(false)

  /** Duração e reprodução via <audio> (Opus/OGG do WhatsApp — Web Audio decode falha com frequência). */
  useEffect(() => {
    const a = audioRef.current
    if (!a) return

    let cancelled = false
    let raf = 0
    setPlaying(false)
    setCurrent(0)
    setDuration(0)
    setBars(null)
    setDecodeErr(false)
    setElementErr(false)

    a.pause()
    a.src = src
    a.preload = 'auto'

    const syncDuration = () => {
      if (cancelled) return
      let d = a.duration
      if (!Number.isFinite(d) || d <= 0 || d === Number.POSITIVE_INFINITY) {
        try {
          if (a.seekable && a.seekable.length > 0) {
            const end = a.seekable.end(a.seekable.length - 1)
            if (Number.isFinite(end) && end > 0) d = end
          }
        } catch {
          /* ignore */
        }
      }
      if (Number.isFinite(d) && d > 0 && !Number.isNaN(d) && d !== Number.POSITIVE_INFINITY) {
        setDuration(d)
      }
    }

    const onTime = () => setCurrent(a.currentTime)
    const onEnd = () => {
      setPlaying(false)
      setCurrent(0)
      a.currentTime = 0
    }

    const pumpDuration = () => {
      if (cancelled) return
      syncDuration()
      const d = a.duration
      const bad = !Number.isFinite(d) || d <= 0 || d === Number.POSITIVE_INFINITY
      if (bad && !a.paused) {
        raf = requestAnimationFrame(pumpDuration)
      }
    }

    const onPlaying = () => {
      syncDuration()
      pumpDuration()
    }

    const onPauseEl = () => {
      cancelAnimationFrame(raf)
      raf = 0
    }

    const onAudioError = () => {
      if (import.meta.env.DEV && a.error) {
        // MEDIA_ERR_ABORTED=1, NETWORK=2, DECODE=3, SRC_NOT_SUPPORTED=4
        console.warn('[VoiceNotePlayer] audio error', {
          code: a.error.code,
          message: a.error.message,
        })
      }
      if (!cancelled) setElementErr(true)
    }

    a.addEventListener('loadedmetadata', syncDuration)
    a.addEventListener('loadeddata', syncDuration)
    a.addEventListener('progress', syncDuration)
    a.addEventListener('durationchange', syncDuration)
    a.addEventListener('canplay', syncDuration)
    a.addEventListener('canplaythrough', syncDuration)
    a.addEventListener('playing', onPlaying)
    a.addEventListener('pause', onPauseEl)
    a.addEventListener('timeupdate', onTime)
    a.addEventListener('ended', onEnd)
    a.addEventListener('error', onAudioError)
    a.load()

    /** Forma de onda opcional — não altera duração se falhar (comum com OGG Opus). */
    ;(async () => {
      try {
        const res = await fetch(src)
        const buf = await res.arrayBuffer()
        if (cancelled || buf.byteLength < 64) return
        const ctx = new AudioContext()
        try {
          const audioBuffer = await ctx.decodeAudioData(buf.slice(0))
          if (cancelled) return
          const d = audioBuffer.duration
          if (Number.isFinite(d) && d > 0) {
            setDuration((prev) => (prev > 0.05 ? prev : d))
          }
          const channel = audioBuffer.getChannelData(0)
          const len = channel.length
          const block = Math.max(1, Math.floor(len / BAR_COUNT))
          const peaks: number[] = []
          for (let i = 0; i < BAR_COUNT; i++) {
            const start = i * block
            const end = Math.min(start + block, len)
            let sum = 0
            for (let j = start; j < end; j++) {
              const v = channel[j]
              sum += v * v
            }
            const rms = Math.sqrt(sum / (end - start))
            peaks.push(rms)
          }
          const max = Math.max(...peaks, 1e-9)
          setBars(peaks.map((p) => Math.max(0.14, Math.min(1, p / max))))
        } finally {
          await ctx.close().catch(() => {})
        }
      } catch {
        if (!cancelled) {
          setDecodeErr(true)
          setBars(fakeWaveBars())
        }
      }
    })()

    return () => {
      cancelled = true
      cancelAnimationFrame(raf)
      a.removeEventListener('loadedmetadata', syncDuration)
      a.removeEventListener('loadeddata', syncDuration)
      a.removeEventListener('progress', syncDuration)
      a.removeEventListener('durationchange', syncDuration)
      a.removeEventListener('canplay', syncDuration)
      a.removeEventListener('canplaythrough', syncDuration)
      a.removeEventListener('playing', onPlaying)
      a.removeEventListener('pause', onPauseEl)
      a.removeEventListener('timeupdate', onTime)
      a.removeEventListener('ended', onEnd)
      a.removeEventListener('error', onAudioError)
      a.pause()
      a.removeAttribute('src')
      a.load()
    }
  }, [src])

  const dur = duration > 0 ? duration : 0
  const progress = dur > 0 ? Math.min(1, current / dur) : 0

  const seekFromPointer = useCallback(
    (clientX: number) => {
      const a = audioRef.current
      const box = waveRef.current
      if (!a || !box || dur <= 0) return
      const rect = box.getBoundingClientRect()
      const ratio = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width))
      a.currentTime = ratio * dur
      setCurrent(a.currentTime)
    },
    [dur],
  )

  const toggle = () => {
    const a = audioRef.current
    if (!a) return
    if (playing) {
      a.pause()
      setPlaying(false)
    } else {
      void a
        .play()
        .then(() => setPlaying(true))
        .catch(() => setPlaying(false))
    }
  }

  const displayBars = bars ?? Array(BAR_COUNT).fill(0.35)

  return (
    <div
      className={cn(
        'flex items-center gap-2 min-w-[240px] max-w-[min(100%,320px)] py-0.5',
        outbound ? 'text-white' : 'text-text-primary',
      )}
    >
      <button
        type="button"
        onClick={toggle}
        className={cn(
          'shrink-0 w-11 h-11 rounded-full flex items-center justify-center transition-colors',
          outbound ? 'bg-white/25 hover:bg-white/35' : 'bg-primary/15 hover:bg-primary/25 text-primary',
        )}
        aria-label={playing ? 'Pausar' : 'Reproduzir'}
      >
        {playing ? <Pause className="size-5" /> : <Play className="size-5 pl-0.5" />}
      </button>

      <div className="flex-1 min-w-0 flex flex-col gap-1.5">
        <div
          ref={waveRef}
          role="slider"
          tabIndex={0}
          aria-valuemin={0}
          aria-valuemax={Math.round(dur)}
          aria-valuenow={Math.round(current)}
          aria-label="Progresso do áudio"
          className={cn(
            'flex items-end justify-between gap-[2px] h-9 px-0.5 cursor-pointer select-none rounded-md',
            outbound ? 'bg-white/10' : 'bg-primary/8',
            decodeErr && 'opacity-90',
          )}
          onClick={(e) => seekFromPointer(e.clientX)}
          onKeyDown={(e) => {
            if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
              e.preventDefault()
              const a = audioRef.current
              if (!a || dur <= 0) return
              const step = dur / BAR_COUNT
              a.currentTime = Math.max(0, Math.min(dur, a.currentTime + (e.key === 'ArrowRight' ? step : -step)))
              setCurrent(a.currentTime)
            }
          }}
        >
          {displayBars.map((h, i) => {
            const t = (i + 0.5) / displayBars.length
            const played = t <= progress + 0.0001
            const heightPx = 6 + h * 26
            return (
              <div
                key={i}
                className={cn(
                  'w-[2px] sm:w-[3px] rounded-full shrink-0 transition-colors duration-150',
                  played
                    ? outbound
                      ? 'bg-white'
                      : 'bg-primary'
                    : outbound
                      ? 'bg-white/40'
                      : 'bg-primary/35',
                )}
                style={{ height: `${heightPx}px` }}
              />
            )
          })}
        </div>

        <div className="flex items-center justify-end gap-1">
          <span
            className={cn(
              'text-[11px] tabular-nums tracking-tight font-medium',
              outbound ? 'text-white/90' : 'text-text-muted',
            )}
          >
            {fmtTime(current)}
            <span className={cn('mx-0.5', outbound ? 'text-white/50' : 'text-text-muted/80')}>/</span>
            {fmtTime(dur)}
          </span>
        </div>
        {elementErr ? (
          <p className={cn('text-[10px]', outbound ? 'text-white/70' : 'text-text-muted')}>
            Não foi possível reproduzir este áudio no navegador.
          </p>
        ) : null}
      </div>

      <audio ref={audioRef} preload="auto" className="hidden" />
    </div>
  )
}
